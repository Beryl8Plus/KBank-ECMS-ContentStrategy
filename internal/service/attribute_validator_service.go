package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	domainrepo "kbank-ecms/internal/domain/repository"
)

// AttributeValueError describes one RuleAttribute whose value is no longer valid.
type AttributeValueError struct {
	RuleAttributeID uuid.UUID
	AttributeID     uuid.UUID
	ChosenValue     string
	AllowedValues   []string
}

func (e AttributeValueError) Error() string {
	return fmt.Sprintf("rule_attribute %s: value %q not in allowed set %v", e.RuleAttributeID, e.ChosenValue, e.AllowedValues)
}

// InactiveAttributeError describes an attribute that has been deactivated.
type InactiveAttributeError struct {
	AttributeID uuid.UUID
	FieldName   string
}

func (e InactiveAttributeError) Error() string {
	return fmt.Sprintf("attribute %s (%s) is inactive", e.AttributeID, e.FieldName)
}

// AttributeValidatorService validates attribute references and values for a
// DecisionRule. It is the single source of truth for this logic and is reused
// by both the integrity checker job and the Step-2/Step-4 API handlers.
type AttributeValidatorService struct {
	syncRepo domainrepo.AttributeSyncRepository
}

// NewAttributeValidatorService creates a new AttributeValidatorService.
func NewAttributeValidatorService(syncRepo domainrepo.AttributeSyncRepository) *AttributeValidatorService {
	return &AttributeValidatorService{syncRepo: syncRepo}
}

// ValidateAttributesActive checks that all given attribute IDs are still active.
// Returns one InactiveAttributeError per deactivated attribute found.
func (s *AttributeValidatorService) ValidateAttributesActive(
	ctx context.Context,
	attributeIDs []uuid.UUID,
) ([]InactiveAttributeError, error) {
	if len(attributeIDs) == 0 {
		return nil, nil
	}
	attrs, err := s.syncRepo.FindAttributesByIDs(ctx, attributeIDs)
	if err != nil {
		return nil, err
	}
	var errs []InactiveAttributeError
	for _, a := range attrs {
		if !a.IsActive {
			errs = append(errs, InactiveAttributeError{
				AttributeID: a.ID,
				FieldName:   a.FieldName,
			})
		}
	}
	return errs, nil
}

// RuleAttributeInput carries the data needed to validate one cell value.
type RuleAttributeInput struct {
	RuleAttributeID uuid.UUID
	AttributeID     uuid.UUID
	// ValueJSON is the raw JSON of the chosen value (e.g. `"GOLD"` or `{"value":"GOLD"}`).
	ValueJSON []byte
}

// ValidateRuleAttributeValues checks that every chosen value is still present in
// its Attribute's allowed options JSON. Returns one AttributeValueError per violation.
func (s *AttributeValidatorService) ValidateRuleAttributeValues(
	ctx context.Context,
	inputs []RuleAttributeInput,
) ([]AttributeValueError, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// Deduplicate attribute IDs for batch load.
	seen := make(map[uuid.UUID]struct{}, len(inputs))
	attrIDs := make([]uuid.UUID, 0, len(inputs))
	for _, inp := range inputs {
		if _, ok := seen[inp.AttributeID]; !ok {
			seen[inp.AttributeID] = struct{}{}
			attrIDs = append(attrIDs, inp.AttributeID)
		}
	}

	attrs, err := s.syncRepo.FindAttributesByIDs(ctx, attrIDs)
	if err != nil {
		return nil, err
	}
	attrMap := make(map[uuid.UUID]*struct {
		allowed map[string]struct{}
	}, len(attrs))
	for _, a := range attrs {
		allowed, parseErr := a.ValidOptions()
		if parseErr != nil {
			// Log-worthy but non-fatal; skip constraint check for this attribute.
			continue
		}
		attrMap[a.ID] = &struct{ allowed map[string]struct{} }{allowed: allowed}
	}

	var errs []AttributeValueError
	for _, inp := range inputs {
		entry, ok := attrMap[inp.AttributeID]
		if !ok || entry.allowed == nil {
			// Attribute not found or has no options constraint.
			continue
		}

		var chosen string
		if jsonErr := json.Unmarshal(inp.ValueJSON, &chosen); jsonErr != nil {
			// Value is not a plain string — cannot validate format.
			continue
		}

		if _, valid := entry.allowed[chosen]; !valid {
			errs = append(errs, AttributeValueError{
				RuleAttributeID: inp.RuleAttributeID,
				AttributeID:     inp.AttributeID,
				ChosenValue:     chosen,
				AllowedValues:   mapKeys(entry.allowed),
			})
		}
	}
	return errs, nil
}

func mapKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
