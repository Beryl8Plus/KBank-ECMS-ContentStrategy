package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"kbank-ecms/internal/domain/entity"
)

// All previously-here lead-enrichment tests covered behavior of
// enrichLeadOffering, which was removed alongside the per-datasource
// cache + UUID-key transform refactor. resolveUserAttrs no longer touches
// lead_offering at all; SALES_TARGET lead enrichment is exercised by
// TestFetchLeadsForRequest_* in cms_delivery_lead_test.go.
//
// The tests below target the small helpers that fell below the per-package
// 75% coverage gate: parseRawEnvelope, lookupSchemaFields, lookupAttributeUUIDs.

// ─────────────────────────────────────────────────────────────────────────────
// parseRawEnvelope
// ─────────────────────────────────────────────────────────────────────────────

// TestParseRawEnvelope_EmptyBody — both nil and zero-length bytes must yield
// an empty map without attempting JSON decode.
func TestParseRawEnvelope_EmptyBody(t *testing.T) {
	t.Parallel()

	assert.Empty(t, parseRawEnvelope(nil))
	assert.Empty(t, parseRawEnvelope([]byte{}))
}

// TestParseRawEnvelope_InvalidJSON — malformed envelopes must self-heal to
// empty, never panic. The cache TTL will refresh the entry on the next call.
func TestParseRawEnvelope_InvalidJSON(t *testing.T) {
	t.Parallel()

	assert.Empty(t, parseRawEnvelope([]byte("not-json")))
	assert.Empty(t, parseRawEnvelope([]byte(`{"results":not-an-array}`)))
}

// TestParseRawEnvelope_OnlySuccessKept — envelope-level metadata is dropped
// and only success rows with a present data field + non-empty datasource
// survive. Note: emptiness is checked at the raw-byte level, so an explicit
// `"data":{}` survives (len("{}") == 2) — only an absent data field is
// dropped via the len==0 branch.
func TestParseRawEnvelope_OnlySuccessKept(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"cis_id":"123",
		"results":[
			{"datasource":"a","status":"success","data":{"x":1}},
			{"datasource":"b","status":"not_found","data":null},
			{"datasource":"c","status":"error","data":{"y":2}},
			{"datasource":"d","status":"success"},
			{"datasource":"","status":"success","data":{"z":3}},
			{"datasource":"e","status":"success","data":{"e":true}}
		],
		"total_sources_queried":6,
		"total_sources_success":3
	}`)

	got := parseRawEnvelope(body)
	require.Len(t, got, 2, "only success rows with present data + datasource should remain")
	assert.JSONEq(t, `{"x":1}`, string(got["a"]))
	assert.JSONEq(t, `{"e":true}`, string(got["e"]))

	for _, ds := range []string{"b", "c", "d", ""} {
		_, ok := got[ds]
		assert.False(t, ok, "datasource %q must be filtered out", ds)
	}
}

// TestParseRawEnvelope_NoResults — well-formed envelope with an empty results
// array yields an empty (but non-nil) map.
func TestParseRawEnvelope_NoResults(t *testing.T) {
	t.Parallel()

	got := parseRawEnvelope([]byte(`{"results":[]}`))
	require.NotNil(t, got)
	assert.Empty(t, got)
}

// ─────────────────────────────────────────────────────────────────────────────
// lookupSchemaFields
// ─────────────────────────────────────────────────────────────────────────────

// stubSchemaRegistryRepo returns a fixed (row, err) pair on GetByID and
// counts calls so cache-hit assertions are possible.
type stubSchemaRegistryRepo struct {
	row   *entity.CLENSchemaRegistry
	err   error
	calls atomic.Int32
}

func (s *stubSchemaRegistryRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.CLENSchemaRegistry, error) {
	s.calls.Add(1)
	return s.row, s.err
}

func TestLookupSchemaFields_NilIDShortCircuits(t *testing.T) {
	t.Parallel()

	stub := &stubSchemaRegistryRepo{}
	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, stub, nil)

	fields, err := svc.lookupSchemaFields(context.Background(), uuid.Nil)
	require.NoError(t, err)
	assert.Nil(t, fields)
	assert.Equal(t, int32(0), stub.calls.Load(), "uuid.Nil must short-circuit before hitting the repo")
}

func TestLookupSchemaFields_NoRepoShortCircuits(t *testing.T) {
	t.Parallel()

	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, nil, nil)
	fields, err := svc.lookupSchemaFields(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, fields, "schemaRegistryRepo unset must short-circuit")
}

func TestLookupSchemaFields_RepoErrorPropagated(t *testing.T) {
	t.Parallel()

	stub := &stubSchemaRegistryRepo{err: errors.New("db down")}
	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, stub, nil)

	fields, err := svc.lookupSchemaFields(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Nil(t, fields)
}

func TestLookupSchemaFields_RowNotFoundCachesEmpty(t *testing.T) {
	t.Parallel()

	stub := &stubSchemaRegistryRepo{row: nil}
	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, stub, nil)

	id := uuid.New()
	fields, err := svc.lookupSchemaFields(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, fields)

	// Repeat: must hit the per-service cache, not the repo.
	_, err = svc.lookupSchemaFields(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, int32(1), stub.calls.Load(),
		"missing rows must be cached as empty so identical lookups don't re-query")
}

func TestLookupSchemaFields_EmptyDefinitionCachesEmpty(t *testing.T) {
	t.Parallel()

	stub := &stubSchemaRegistryRepo{row: &entity.CLENSchemaRegistry{
		BaseModel:        entity.BaseModel{ID: uuid.New()},
		SchemaDefinition: datatypes.JSON([]byte{}),
	}}
	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, stub, nil)

	fields, err := svc.lookupSchemaFields(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, fields, "empty SchemaDefinition must yield an empty field set")
}

func TestLookupSchemaFields_ParsedAndCached(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	stub := &stubSchemaRegistryRepo{row: fakeSchema(id, "ds", "a", "b", "c")}
	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, stub, nil)

	first, err := svc.lookupSchemaFields(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, first, 3)
	for _, k := range []string{"a", "b", "c"} {
		_, ok := first[k]
		assert.True(t, ok, "missing parsed field %q", k)
	}

	second, err := svc.lookupSchemaFields(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, first, second, "same UUID must return the cached map")
	assert.Equal(t, int32(1), stub.calls.Load(), "cached lookup must not re-query the repo")
}

// ─────────────────────────────────────────────────────────────────────────────
// lookupAttributeUUIDs
// ─────────────────────────────────────────────────────────────────────────────

// stubAttributeRepo returns a fixed (rows, err) pair on ListByTableSourceName
// and counts calls so cache-hit assertions are possible.
type stubAttributeRepo struct {
	rows  []*entity.Attribute
	err   error
	calls atomic.Int32
}

func (s *stubAttributeRepo) CreateAttribute(_ context.Context, _ *entity.Attribute) error {
	return nil
}
func (s *stubAttributeRepo) GetAttributeByID(_ context.Context, _ uuid.UUID) (*entity.Attribute, error) {
	return nil, nil
}
func (s *stubAttributeRepo) ListAttributesPaginated(_ context.Context, _, _ int) ([]*entity.Attribute, int64, error) {
	return nil, 0, nil
}
func (s *stubAttributeRepo) ListByTableSourceName(_ context.Context, _ string) ([]*entity.Attribute, error) {
	s.calls.Add(1)
	return s.rows, s.err
}
func (s *stubAttributeRepo) UpdateAttribute(_ context.Context, _ *entity.Attribute) error {
	return nil
}
func (s *stubAttributeRepo) DeleteAttribute(_ context.Context, _ uuid.UUID) error { return nil }

func TestLookupAttributeUUIDs_EmptyDatasourceShortCircuits(t *testing.T) {
	t.Parallel()

	stub := &stubAttributeRepo{}
	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, nil, stub)

	got, err := svc.lookupAttributeUUIDs(context.Background(), "")
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int32(0), stub.calls.Load())
}

func TestLookupAttributeUUIDs_NoRepoShortCircuits(t *testing.T) {
	t.Parallel()

	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, nil, nil)
	got, err := svc.lookupAttributeUUIDs(context.Background(), "ds")
	require.NoError(t, err)
	assert.Nil(t, got, "attributeRepo unset must short-circuit")
}

func TestLookupAttributeUUIDs_RepoErrorPropagated(t *testing.T) {
	t.Parallel()

	stub := &stubAttributeRepo{err: errors.New("db down")}
	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, nil, stub)

	got, err := svc.lookupAttributeUUIDs(context.Background(), "ds")
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestLookupAttributeUUIDs_BuildsMappingAndSkipsBlanks(t *testing.T) {
	t.Parallel()

	id1 := uuid.New()
	id2 := uuid.New()
	stub := &stubAttributeRepo{rows: []*entity.Attribute{
		{BaseModel: entity.BaseModel{ID: id1}, FieldName: "wlth_seg_cd"},
		nil, // must be skipped without panicking
		{BaseModel: entity.BaseModel{ID: uuid.New()}, FieldName: ""}, // blank field name dropped
		{BaseModel: entity.BaseModel{ID: id2}, FieldName: "avg_aum_6_mo"},
	}}
	svc := newSvcWithCustomerProfile(&mockCacheRepo{}, nil, CustomerProfileEnrichConfig{}, nil, stub)

	got, err := svc.lookupAttributeUUIDs(context.Background(), "ds")
	require.NoError(t, err)
	require.Len(t, got, 2, "nil rows and blank FieldName entries must be dropped")
	assert.Equal(t, id1, got["wlth_seg_cd"])
	assert.Equal(t, id2, got["avg_aum_6_mo"])

	// Second call hits the per-service cache.
	_, err = svc.lookupAttributeUUIDs(context.Background(), "ds")
	require.NoError(t, err)
	assert.Equal(t, int32(1), stub.calls.Load(), "cached lookup must not re-query the repo")
}
