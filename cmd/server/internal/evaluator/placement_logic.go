package evaluator

import (
	"encoding/json"
	"time"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
)

// BuildPlacementLogicEntries constructs one ContentResult per variation of a
// DecisionRule. Each entry carries the variation's expected values in its
// Conditions, enabling delivery-time evaluation against live user attributes.
//
// If the rule has no variations, a single entry is returned with the rule's
// base score and conditions without expected values (no-op at delivery time).
func BuildPlacementLogicEntries(
	rule entity.DecisionRule,
	sched *entity.Schedule,
	source string,
	campaign *dto.Campaign,
) []dto.ContentResult {
	// Support type Mass, that no rules or variations, just a single content path and score.
	if len(rule.Rules) == 0 {
		// No variations — single entry with base score, empty expected values.
		entry := buildLogicEntry(rule, sched, nil, rule.Score, source, campaign, nil)
		return []dto.ContentResult{entry}
	}

	results := make([]dto.ContentResult, 0, len(rule.Rules))
	for _, v := range sortedVariations(rule.Rules) {
		expectedValues := make(map[string]json.RawMessage, len(v.RuleAttributes))
		for _, ra := range v.RuleAttributes {
			expectedValues[ra.AttributeID.String()] = json.RawMessage(ra.Value)
		}
		varName := v.VariationName
		entry := buildLogicEntry(rule, sched, &varName, float64(v.Score), source, campaign, expectedValues)
		results = append(results, entry)
	}
	return results
}

// buildLogicEntry constructs a single ContentResult with populated Conditions.
func buildLogicEntry(
	rule entity.DecisionRule,
	sched *entity.Schedule,
	variation *string,
	score float64,
	source string,
	campaign *dto.Campaign,
	expectedValues map[string]json.RawMessage,
) dto.ContentResult {
	logicHash, _ := GenerateLogicHash(rule.RuleConditions, expectedValues)
	logicExpr := BuildLogicExpression(rule.RuleConditions, expectedValues)

	conditions := make([]dto.LogicCondition, 0, len(rule.RuleConditions))
	for _, rc := range rule.RuleConditions {
		lc := dto.LogicCondition{
			ConditionID:       rc.ID.String(),
			AttributeID:       rc.AttributeID.String(),
			LogicalOperator:   string(rc.LogicalOperator),
			ConnectorOperator: string(rc.ConnectorOperator),
			Sequence:          rc.Sequence,
			ExpectedValue:     expectedValues[rc.AttributeID.String()],
		}
		if rc.ParentRuleConditionID != nil {
			lc.ParentConditionID = rc.ParentRuleConditionID.String()
		}
		if rc.Attribute != nil {
			lc.DataType = string(rc.Attribute.DataType)
		}
		conditions = append(conditions, lc)
	}

	return dto.ContentResult{
		DecisionRuleId: sched.DecisionRuleID.String(),
		ContentPath:    rule.ContentPath,
		RuleSetType:    rule.Type.String(),
		Source:         source,
		Score:          score,
		Variation:      variation,
		StartDateTime:  sched.EffectiveFrom.Format(time.RFC3339),
		EndDateTime:    sched.EffectiveUntil.Format(time.RFC3339),
		Campaign:       campaign,
		LogicHash:      logicHash,
		LogicExpr:      logicExpr,
		Conditions:     conditions,
	}
}
