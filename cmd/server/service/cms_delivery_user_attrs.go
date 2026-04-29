package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/logger"
)

func cmsUserAttrsKey(customerType, customerID string) string {
	return "customer_profile:" + customerType + ":" + customerID
}

// attributeSourceCLEN is the SourceSystem marker for attributes that resolve
// from CLEN. Only these attributes participate in CLEN datasource routing.
const attributeSourceCLEN = "CLEN"

// ruleScope captures, per CLEN datasource (TableSourceName), the union of
// fields directly referenced by the rule and the schema-registry ID those
// attributes belong to. The schema ID is later used to look up the master
// dictionary of valid fields for that datasource.
type ruleScope struct {
	schemaID uuid.UUID
	fields   map[string]struct{}
}

// schemaFieldsLookup returns the set of valid field names for a given
// CLENSchemaRegistry ID. Implementations are typically backed by an
// in-memory cache fronting a DB query.
type schemaFieldsLookup func(ctx context.Context, schemaID uuid.UUID) (map[string]struct{}, error)

// extractCLENDataSources derives the CLEN per-datasource queries needed to
// warm the customer profile cache for the given rules. For each datasource a
// rule touches, it looks up the schema's full field dictionary, subtracts
// fields the rule already references (those values come via the rule
// itself), then subtracts fields already in the cache, and emits only the
// remaining "schema-extra, not-yet-cached" fields.
//
// Returns nil when nothing needs fetching (cache already covers schema
// extras), when no rules reference CLEN attributes, or when lookupSchema is
// nil.
func extractCLENDataSources(
	ctx context.Context,
	rules []*entity.DecisionRule,
	cached map[string]json.RawMessage,
	lookupSchema schemaFieldsLookup,
) []domainrepo.CustomerProfileDataSource {
	if len(rules) == 0 || lookupSchema == nil {
		return nil
	}

	// 1. Collect rule's per-datasource fields + remember schemaID per ds.
	needed := map[string]*ruleScope{}
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		for _, c := range rule.RuleConditions {
			collectCLENAttr(c.Attribute, needed)
		}
	}
	if len(needed) == 0 {
		return nil
	}

	// 2. For each datasource, compute (schema − rule) − cache.
	out := make([]domainrepo.CustomerProfileDataSource, 0, len(needed))
	for ds, scope := range needed {
		schemaFields, err := lookupSchema(ctx, scope.schemaID)
		if err != nil || len(schemaFields) == 0 {
			continue
		}
		var dsCached map[string]json.RawMessage
		if blob, ok := cached[ds]; ok && len(blob) > 0 {
			_ = json.Unmarshal(blob, &dsCached)
		}
		missing := make([]string, 0)
		for f := range schemaFields {
			if _, inRule := scope.fields[f]; inRule {
				continue
			}
			if _, inCache := dsCached[f]; inCache {
				continue
			}
			missing = append(missing, f)
		}
		if len(missing) > 0 {
			out = append(out, domainrepo.CustomerProfileDataSource{
				Datasource:     ds,
				RequiredFields: missing,
			})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// collectCLENAttr filters in only CLEN-sourced attributes that have both a
// TableSourceName (CLEN datasource identifier) and FieldName, then records
// the field under its datasource in the needed map. The schemaID of the
// first attribute seen for a datasource wins (attributes mapping to the
// same datasource are expected to share a schema in practice).
func collectCLENAttr(a *entity.Attribute, into map[string]*ruleScope) {
	if a == nil || a.SourceSystem != attributeSourceCLEN || a.TableSourceName == "" || a.FieldName == "" {
		return
	}
	scope, ok := into[a.TableSourceName]
	if !ok {
		scope = &ruleScope{schemaID: a.ClenSchemaRegistryID, fields: map[string]struct{}{}}
		into[a.TableSourceName] = scope
	}
	scope.fields[a.FieldName] = struct{}{}
}

// lookupSchemaFields returns the set of fields declared in the
// SchemaDefinition of the given clen_schema_registry row, with results
// cached per service instance to avoid repeated DB hits and JSON parses.
// Returns (nil, nil) for nil/zero IDs or rows that don't exist.
func (s *CMSDeliveryService) lookupSchemaFields(ctx context.Context, id uuid.UUID) (map[string]struct{}, error) {
	if id == uuid.Nil || s.schemaRegistryRepo == nil {
		return nil, nil
	}
	if v, ok := s.schemaFieldsCache.Load(id); ok {
		fields, _ := v.(map[string]struct{})
		return fields, nil
	}
	row, err := s.schemaRegistryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil || len(row.SchemaDefinition) == 0 {
		s.schemaFieldsCache.Store(id, map[string]struct{}{})
		return map[string]struct{}{}, nil
	}
	fields := parseSchemaDefinitionFields(row.SchemaDefinition)
	s.schemaFieldsCache.Store(id, fields)
	return fields, nil
}

// parseSchemaDefinitionFields extracts field names from the JSON-Schema-like
// payload stored in clen_schema_registry.SCHEMA_DEFINITION. Only the top
// level "properties" object is read; values (type/constraints) are ignored.
func parseSchemaDefinitionFields(raw []byte) map[string]struct{} {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil || len(schema.Properties) == 0 {
		return map[string]struct{}{}
	}
	fields := make(map[string]struct{}, len(schema.Properties))
	for k := range schema.Properties {
		fields[k] = struct{}{}
	}
	return fields
}

// lookupAttributeUUIDs returns a field-name → attribute-UUID map for the
// given CLEN datasource (TableSourceName). Loaded lazily from the
// attributes table and cached per service instance to avoid repeated DB
// hits when transforming the cached customer profile blob.
func (s *CMSDeliveryService) lookupAttributeUUIDs(ctx context.Context, datasource string) (map[string]uuid.UUID, error) {
	if datasource == "" || s.attributeRepo == nil {
		return nil, nil
	}
	if v, ok := s.attrFieldToUUIDByDsCache.Load(datasource); ok {
		mapping, _ := v.(map[string]uuid.UUID)
		return mapping, nil
	}
	rows, err := s.attributeRepo.ListByTableSourceName(ctx, datasource)
	if err != nil {
		return nil, err
	}
	mapping := make(map[string]uuid.UUID, len(rows))
	for _, a := range rows {
		if a == nil || a.FieldName == "" {
			continue
		}
		mapping[a.FieldName] = a.ID
	}
	s.attrFieldToUUIDByDsCache.Store(datasource, mapping)
	return mapping, nil
}

// transformToUUIDKeyed walks the per-datasource cached blob and produces a
// flat UUID-keyed attribute map suitable for the rule evaluator. Each
// (datasource, fieldName) pair is resolved to its Attribute.ID via
// lookupAttributeUUIDs; pairs without a matching attribute row are dropped.
func (s *CMSDeliveryService) transformToUUIDKeyed(
	ctx context.Context,
	cached map[string]json.RawMessage,
) map[string]json.RawMessage {
	if len(cached) == 0 || s.attributeRepo == nil {
		return map[string]json.RawMessage{}
	}
	out := make(map[string]json.RawMessage)
	for ds, blob := range cached {
		if len(blob) == 0 {
			continue
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(blob, &fields); err != nil {
			continue
		}
		fieldToUUID, err := s.lookupAttributeUUIDs(ctx, ds)
		if err != nil || len(fieldToUUID) == 0 {
			continue
		}
		for fieldName, value := range fields {
			id, ok := fieldToUUID[fieldName]
			if !ok {
				continue
			}
			out[id.String()] = value
		}
	}
	return out
}

// resolveUserAttrs reads the per-datasource customer-profile map from Redis
// at key customer_profile:{customerType}:{customerID}, computes the delta between what the rules need and
// what is already cached, fetches only the missing fields from CLEN, and
// returns the merged map. Keys are CLEN datasource names (e.g.
// "cst_info_prfl_dly"); values are JSON objects of {field: value}.
//
// rules drives the per-request enrichment scope — when nil/empty, no CLEN
// call is made and only the cached blob (possibly empty) is returned.
//
// Note: lead-offering and externally-provided UUID attrs are no longer merged
// into this cache key — handle those separately if needed.
func (s *CMSDeliveryService) resolveUserAttrs(
	ctx context.Context,
	customerType string,
	customerId string,
	rules []*entity.DecisionRule,
) (map[string]json.RawMessage, error) {
	if s.cacheRepo == nil || customerId == "" {
		return map[string]json.RawMessage{}, nil
	}

	cacheKey := cmsUserAttrsKey(customerType, customerId)
	var cached map[string]json.RawMessage
	raw, err := s.cacheRepo.Get(ctx, cacheKey)
	if err == nil {
		if err := json.Unmarshal([]byte(raw), &cached); err != nil {
			return nil, fmt.Errorf("decode user attrs from %q: %w", cacheKey, err)
		}
	}
	if cached == nil {
		cached = map[string]json.RawMessage{}
	}

	// Schema-driven warm fetch — for each datasource a rule touches, look up
	// the schema's full field dictionary and request the schema-extra fields
	// (those not referenced by the rule) that are not yet in the cache.
	// Short-circuits without calling CLEN when the cache already covers
	// every schema-extra field.
	if s.customerProfileRepo != nil {
		if missing := extractCLENDataSources(ctx, rules, cached, s.lookupSchemaFields); len(missing) > 0 {
			s.enrichCustomerProfile(ctx, customerType, customerId, cached, missing)
		}
	}

	// Transform the per-datasource cached shape into the flat UUID-keyed
	// shape the rule evaluator expects. Field names are resolved to
	// Attribute.ID via lookupAttributeUUIDs (per-datasource, cached).
	resolved := s.transformToUUIDKeyed(ctx, cached)

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("Loaded %d datasources (%d attrs after transform) from %q", len(cached), len(resolved), cacheKey),
	})

	return resolved, nil
}

// enrichCustomerProfile fetches the requested CLEN data sources, then merges
// each successful result into attrs at field level (preserving any existing
// fields under the same datasource), and persists the updated blob back to
// Redis. Upstream errors are logged but swallowed — the personalized-content
// request must not fail because of a CLEN outage.
//
// sources is the per-request datasource/field list — typically the delta
// computed by extractCLENDataSources.
func (s *CMSDeliveryService) enrichCustomerProfile(
	ctx context.Context,
	customerType string,
	cisID string,
	attrs map[string]json.RawMessage,
	sources []domainrepo.CustomerProfileDataSource,
) {
	if len(sources) == 0 {
		return
	}

	raw, err := s.customerProfileRepo.GetCustomerProfileRaw(ctx, domainrepo.CustomerProfileQuery{
		CisID:       cisID,
		DataSources: sources,
	})
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "WARN",
			Message: "CLEN customer profile fetch failed for cis " + cisID + ": " + err.Error(),
		})
		return
	}
	if raw == nil || raw.StatusCode < 200 || raw.StatusCode >= 300 || len(raw.Body) == 0 {
		return
	}

	var parsed struct {
		Results []struct {
			Datasource string          `json:"datasource"`
			Status     string          `json:"status"`
			Data       json.RawMessage `json:"data"`
		} `json:"results"`
	}
	if uErr := json.Unmarshal(raw.Body, &parsed); uErr != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-DELIVERY",
			Level:   "WARN",
			Message: "decode CLEN customer profile body for cis " + cisID + ": " + uErr.Error(),
		})
		return
	}

	// Field-level merge — preserve existing fields under each datasource and
	// only add/overwrite fields that the upstream returned.
	mutated := false
	for _, r := range parsed.Results {
		if r.Status != "success" || len(r.Data) == 0 {
			continue
		}
		var incoming map[string]json.RawMessage
		if err := json.Unmarshal(r.Data, &incoming); err != nil {
			continue
		}
		var existing map[string]json.RawMessage
		if blob, ok := attrs[r.Datasource]; ok && len(blob) > 0 {
			_ = json.Unmarshal(blob, &existing)
		}
		if existing == nil {
			existing = make(map[string]json.RawMessage, len(incoming))
		}
		for k, v := range incoming {
			existing[k] = v
		}
		merged, mErr := json.Marshal(existing)
		if mErr != nil {
			continue
		}
		attrs[r.Datasource] = merged
		mutated = true
	}

	if !mutated {
		return
	}

	if buf, bErr := json.Marshal(attrs); bErr == nil {
		ttl := s.customerProfileEnrich.CacheTTL
		if ttl <= 0 {
			ttl = 1 * time.Hour
		}
		_ = s.cacheRepo.Set(ctx, cmsUserAttrsKey(customerType, cisID), string(buf), ttl)
	}
}
