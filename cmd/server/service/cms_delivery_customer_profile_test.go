package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
)

type fakeCustomerProfileRepo struct {
	profile   *entity.CustomerProfile
	err       error
	rawResp   *domainrepo.RawCustomerProfileResponse
	rawErr    error
	calls     int
	rawCalls  int
	lastQuery domainrepo.CustomerProfileQuery
}

func (f *fakeCustomerProfileRepo) GetCustomerProfile(_ context.Context, q domainrepo.CustomerProfileQuery) (*entity.CustomerProfile, error) {
	f.calls++
	f.lastQuery = q
	return f.profile, f.err
}

func (f *fakeCustomerProfileRepo) GetCustomerProfileRaw(_ context.Context, q domainrepo.CustomerProfileQuery) (*domainrepo.RawCustomerProfileResponse, error) {
	f.rawCalls++
	f.lastQuery = q
	if f.rawResp != nil || f.rawErr != nil {
		return f.rawResp, f.rawErr
	}
	return &domainrepo.RawCustomerProfileResponse{StatusCode: 200, Body: []byte("{}")}, nil
}

// newSvcWithCustomerProfile builds a CMSDeliveryService wired with a customer-profile repo + enrich config.
// Pass schemaRepo / attrRepo as nil when the test does not exercise schema-driven enrichment or UUID-key transform.
func newSvcWithCustomerProfile(
	cacheRepo *mockCacheRepo,
	cpRepo domainrepo.CustomerProfileRepository,
	cpEnrich CustomerProfileEnrichConfig,
	schemaRepo domainrepo.CLENSchemaRegistryRepository,
	attrRepo domainrepo.AttributeRepository,
) *CMSDeliveryService {
	return NewCMSDeliveryService(
		cacheRepo, &mockOccurrenceRepo{}, &mockDecisionRuleRepo{},
		nil, nil, time.Hour, 0,
		nil,
		cpRepo, cpEnrich,
		schemaRepo,
		attrRepo,
	)
}

// fakeAttributeRepo is an in-memory stub for AttributeRepository, focused on
// ListByTableSourceName which is the only method used by the delivery
// transform path.
type fakeAttributeRepo struct {
	byDatasource map[string][]*entity.Attribute
}

func (f *fakeAttributeRepo) CreateAttribute(_ context.Context, _ *entity.Attribute) error {
	return nil
}

func (f *fakeAttributeRepo) GetAttributeByID(_ context.Context, _ uuid.UUID) (*entity.Attribute, error) {
	return nil, nil
}

func (f *fakeAttributeRepo) ListAttributesPaginated(_ context.Context, _, _ int) ([]*entity.Attribute, int64, error) {
	return nil, 0, nil
}

func (f *fakeAttributeRepo) ListByTableSourceName(_ context.Context, ds string) ([]*entity.Attribute, error) {
	if f == nil || f.byDatasource == nil {
		return nil, nil
	}
	return f.byDatasource[ds], nil
}

func (f *fakeAttributeRepo) UpdateAttribute(_ context.Context, _ *entity.Attribute) error {
	return nil
}

func (f *fakeAttributeRepo) DeleteAttribute(_ context.Context, _ uuid.UUID) error {
	return nil
}

// fakeSchemaRegistryRepo is an in-memory stub for CLENSchemaRegistryRepository.
// Tests pre-populate `byID` with the schema rows they need.
type fakeSchemaRegistryRepo struct {
	byID map[uuid.UUID]*entity.CLENSchemaRegistry
}

func (f *fakeSchemaRegistryRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.CLENSchemaRegistry, error) {
	if f == nil || f.byID == nil {
		return nil, nil
	}
	return f.byID[id], nil
}

// ruleWithCLENAttrs builds a minimal *entity.DecisionRule whose RuleConditions
// reference CLEN-sourced attributes — used by tests that need to exercise the
// rule-driven CLEN delta-fetch path. All attributes share schemaID so the
// schema lookup resolves to a single registry row in tests.
func ruleWithCLENAttrs(datasource string, schemaID uuid.UUID, fields ...string) *entity.DecisionRule {
	conditions := make([]entity.RuleCondition, 0, len(fields))
	for _, f := range fields {
		conditions = append(conditions, entity.RuleCondition{
			Attribute: &entity.Attribute{
				ClenSchemaRegistryID: schemaID,
				SourceSystem:         "CLEN",
				TableSourceName:      datasource,
				FieldName:            f,
			},
		})
	}
	return &entity.DecisionRule{RuleConditions: conditions}
}

// fakeSchema returns a populated CLENSchemaRegistry with a JSON Schema body
// listing the given field names.
func fakeSchema(id uuid.UUID, schemaName string, fields ...string) *entity.CLENSchemaRegistry {
	props := make(map[string]any, len(fields))
	for _, f := range fields {
		props[f] = map[string]string{"type": "string"}
	}
	body, _ := json.Marshal(map[string]any{"type": "object", "properties": props})
	return &entity.CLENSchemaRegistry{
		BaseModel:        entity.BaseModel{ID: id},
		SchemaName:       schemaName,
		Version:          "1.0.0",
		SchemaDefinition: datatypes.JSON(body),
		IsActive:         true,
	}
}

// TestResolveUserAttrs_CustomerProfileEnrichment_CacheMiss verifies that on
// cache miss, the (schema − rule) − cache delta is sent to CLEN and the
// result is merged + persisted under each datasource.
func TestResolveUserAttrs_CustomerProfileEnrichment_CacheMiss(t *testing.T) {
	t.Parallel()

	var setCalls int
	var setKey, setVal string
	var setTTL time.Duration
	cache := &mockCacheRepo{
		getFn: func(_ context.Context, _ string) (string, error) { return "", errors.New("miss") },
		setFn: func(_ context.Context, k, v string, ttl time.Duration) error {
			setCalls++
			setKey, setVal, setTTL = k, v, ttl
			return nil
		},
	}
	// Schema declares 3 fields; rule covers only wlth_seg_cd, so CLEN is
	// asked for the remaining two (wlth_size, avg_aum_6_mo).
	schemaID := uuid.New()
	wlthSegCdUUID := uuid.New()
	wlthSizeUUID := uuid.New()
	avgAum6MoUUID := uuid.New()
	rawBody := `{"cis_id":"123","results":[{"datasource":"cst_info_prfl_dly","status":"success","data":{"wlth_size":5000000,"avg_aum_6_mo":4800000},"error_message":null}],"total_sources_queried":1,"total_sources_success":1}`
	repo := &fakeCustomerProfileRepo{
		rawResp: &domainrepo.RawCustomerProfileResponse{
			StatusCode: 200,
			Body:       []byte(rawBody),
		},
	}
	schemaRepo := &fakeSchemaRegistryRepo{byID: map[uuid.UUID]*entity.CLENSchemaRegistry{
		schemaID: fakeSchema(schemaID, "cst_info_prfl_dly", "wlth_seg_cd", "wlth_size", "avg_aum_6_mo"),
	}}
	attrRepo := &fakeAttributeRepo{byDatasource: map[string][]*entity.Attribute{
		"cst_info_prfl_dly": {
			{BaseModel: entity.BaseModel{ID: wlthSegCdUUID}, FieldName: "wlth_seg_cd"},
			{BaseModel: entity.BaseModel{ID: wlthSizeUUID}, FieldName: "wlth_size"},
			{BaseModel: entity.BaseModel{ID: avgAum6MoUUID}, FieldName: "avg_aum_6_mo"},
		},
	}}
	enrich := CustomerProfileEnrichConfig{CacheTTL: 30 * time.Second}
	svc := newSvcWithCustomerProfile(cache, repo, enrich, schemaRepo, attrRepo)

	rules := []*entity.DecisionRule{ruleWithCLENAttrs("cst_info_prfl_dly", schemaID, "wlth_seg_cd")}
	attrs, err := svc.resolveUserAttrs(context.Background(), "CIS_ID", "123", rules)
	require.NoError(t, err)
	require.Equal(t, 1, repo.rawCalls)

	// Resolved attrs are flat UUID-keyed (the shape the evaluator expects).
	// The schema-extras fetched from CLEN appear here; the rule field's
	// value isn't fetched, so its UUID should be absent.
	assert.JSONEq(t, `5000000`, string(attrs[wlthSizeUUID.String()]),
		"wlth_size value must be keyed by its attribute UUID")
	assert.JSONEq(t, `4800000`, string(attrs[avgAum6MoUUID.String()]),
		"avg_aum_6_mo value must be keyed by its attribute UUID")
	_, hasRuleField := attrs[wlthSegCdUUID.String()]
	assert.False(t, hasRuleField, "rule field's UUID must not be in the transformed attrs")

	// CLEN must be queried with only the schema-extra fields, never with
	// fields the rule already references.
	require.Len(t, repo.lastQuery.DataSources, 1)
	assert.Equal(t, "cst_info_prfl_dly", repo.lastQuery.DataSources[0].Datasource)
	assert.ElementsMatch(t, []string{"wlth_size", "avg_aum_6_mo"}, repo.lastQuery.DataSources[0].RequiredFields)

	// The Redis cache still stores the nested per-datasource shape
	// (UUID-keyed transform happens only on the returned value).
	assert.Equal(t, 1, setCalls)
	assert.Equal(t, "customer_profile:CIS_ID:123", setKey)
	assert.Equal(t, 30*time.Second, setTTL)
	assert.JSONEq(t, `{"cst_info_prfl_dly":{"wlth_size":5000000,"avg_aum_6_mo":4800000}}`, setVal,
		"cache value must contain only the per-datasource fetched data")
}

// TestResolveUserAttrs_CustomerProfileEnrichment_AlreadyCached verifies that
// CLEN is skipped when every schema-extra field is already in the cache,
// and that the cached values are returned UUID-keyed.
func TestResolveUserAttrs_CustomerProfileEnrichment_AlreadyCached(t *testing.T) {
	t.Parallel()

	// Schema has fields y, w. Rule covers y. Cache already has the
	// remaining schema-extra (w) → no CLEN call needed.
	schemaID := uuid.New()
	yUUID := uuid.New()
	wUUID := uuid.New()
	cached := `{"x":{"y":"z","w":"q"}}`
	cache := &mockCacheRepo{getFn: func(_ context.Context, _ string) (string, error) { return cached, nil }}
	repo := &fakeCustomerProfileRepo{}
	schemaRepo := &fakeSchemaRegistryRepo{byID: map[uuid.UUID]*entity.CLENSchemaRegistry{
		schemaID: fakeSchema(schemaID, "x", "y", "w"),
	}}
	attrRepo := &fakeAttributeRepo{byDatasource: map[string][]*entity.Attribute{
		"x": {
			{BaseModel: entity.BaseModel{ID: yUUID}, FieldName: "y"},
			{BaseModel: entity.BaseModel{ID: wUUID}, FieldName: "w"},
		},
	}}
	svc := newSvcWithCustomerProfile(cache, repo, CustomerProfileEnrichConfig{CacheTTL: time.Minute}, schemaRepo, attrRepo)

	rules := []*entity.DecisionRule{ruleWithCLENAttrs("x", schemaID, "y")}
	attrs, err := svc.resolveUserAttrs(context.Background(), "CIS_ID", "123", rules)
	require.NoError(t, err)
	assert.Equal(t, 0, repo.rawCalls, "cache covers schema-extras — must skip CLEN")
	// Returned attrs are UUID-keyed; both cached fields surface here.
	assert.JSONEq(t, `"z"`, string(attrs[yUUID.String()]))
	assert.JSONEq(t, `"q"`, string(attrs[wUUID.String()]))
}

// TestResolveUserAttrs_CustomerProfileEnrichment_UpstreamError verifies CLEN
// errors are swallowed and the resolved attrs are simply empty.
func TestResolveUserAttrs_CustomerProfileEnrichment_UpstreamError(t *testing.T) {
	t.Parallel()

	schemaID := uuid.New()
	cache := &mockCacheRepo{getFn: func(_ context.Context, _ string) (string, error) { return "", errors.New("miss") }}
	repo := &fakeCustomerProfileRepo{rawErr: errors.New("CLEN down")}
	schemaRepo := &fakeSchemaRegistryRepo{byID: map[uuid.UUID]*entity.CLENSchemaRegistry{
		schemaID: fakeSchema(schemaID, "x", "y", "w"),
	}}
	svc := newSvcWithCustomerProfile(cache, repo, CustomerProfileEnrichConfig{CacheTTL: time.Minute}, schemaRepo, nil)

	rules := []*entity.DecisionRule{ruleWithCLENAttrs("x", schemaID, "y")}
	attrs, err := svc.resolveUserAttrs(context.Background(), "CIS_ID", "123", rules)
	require.NoError(t, err, "CLEN error must not propagate")
	assert.Empty(t, attrs, "resolved attrs must be empty when CLEN fails on cache miss")
}

// TestResolveUserAttrs_CustomerProfileEnrichment_DisabledWhenRulesEmpty
// verifies the enrichment is a no-op when no rules are supplied, even with
// a configured repo and a cache miss.
func TestResolveUserAttrs_CustomerProfileEnrichment_DisabledWhenRulesEmpty(t *testing.T) {
	t.Parallel()

	cache := &mockCacheRepo{getFn: func(_ context.Context, _ string) (string, error) { return "", errors.New("miss") }}
	repo := &fakeCustomerProfileRepo{}
	svc := newSvcWithCustomerProfile(cache, repo, CustomerProfileEnrichConfig{}, nil, nil)

	_, err := svc.resolveUserAttrs(context.Background(), "CIS_ID", "123", nil)
	require.NoError(t, err)
	assert.Equal(t, 0, repo.rawCalls, "no rules → no CLEN call")
}
