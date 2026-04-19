package service

import (
	"context"
	"encoding/json"
	"fmt"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

var cisAttributeFieldToID = map[string]string{
	"cis_num":        "attribute_uuid_1",
	"bsn_size":       "attribute_uuid_2",
	"avg_aum_6_mo":   "attribute_uuid_3",
	"cst_invstr_tp":  "attribute_uuid_4",
	"age_vulner_f":   "attribute_uuid_5",
	"edu_vulner_f":   "attribute_uuid_6",
	"frgnr_vulner_f": "attribute_uuid_7",
	"phys_vulner_f":  "attribute_uuid_8",
}

func cmsUserAttrsKey(cisID string) string {
	return "cis_id:" + cisID
}

func normalizeUserAttrs(raw map[string]json.RawMessage) map[string]json.RawMessage {
	if len(raw) == 0 {
		return raw
	}

	normalized := make(map[string]json.RawMessage, len(raw))

	for key, value := range raw {
		if _, mapped := cisAttributeFieldToID[key]; mapped {
			continue
		}
		normalized[key] = value
	}

	for key, value := range raw {
		attrID, ok := cisAttributeFieldToID[key]
		if !ok {
			continue
		}
		if _, exists := normalized[attrID]; !exists {
			normalized[attrID] = value
		}
	}

	return normalized
}

func mergeUserAttrs(base, override map[string]json.RawMessage) map[string]json.RawMessage {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}

	merged := make(map[string]json.RawMessage, len(base)+len(override))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range override {
		merged[key] = value
	}

	return merged
}

func (s *CMSDeliveryService) resolveUserAttrs(
	ctx context.Context,
	cisID string,
	provided map[string]json.RawMessage,
) (map[string]json.RawMessage, error) {
	normalizedProvided := normalizeUserAttrs(provided)
	if s.cacheRepo == nil || cisID == "" {
		return normalizedProvided, nil
	}

	raw, err := s.cacheRepo.Get(ctx, cmsUserAttrsKey(cisID))
	if err != nil {
		return normalizedProvided, nil
	}

	var cached map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return nil, fmt.Errorf("decode user attrs from %q: %w", cmsUserAttrsKey(cisID), err)
	}

	resolved := mergeUserAttrs(normalizeUserAttrs(cached), normalizedProvided)
	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-DELIVERY",
		Level:   "INFO",
		Message: fmt.Sprintf("Loaded %d user attributes from %q", len(resolved), cmsUserAttrsKey(cisID)),
	})

	return resolved, nil
}
