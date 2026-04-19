package service

import (
	"context"
	"encoding/json"
)

// ContentResult represents a single evaluated content item for a placement.
type Campaign struct {
	Code      string `json:"code"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type ContentResult struct {
	LogicHash      string           `json:"logicHash,omitempty"` // Stable SHA-256 hash; present for logic cache entries
	LogicExpr      string           `json:"logicExpr,omitempty"` // Human-readable expression; present for logic cache entries
	LogicEval      bool             `json:"logicEval,omitempty"` // Evaluation result of the logic expression; present for logic cache entries
	EvaluatedAt    string           `json:"-"`                   // RFC3339 UTC timestamp of evaluation; not stored in cache
	DecisionRuleId string           `json:"decisionRuleId"`
	RuleSetType    string           `json:"ruleSetType"`
	ContentPath    string           `json:"contentPath"`
	Source         string           `json:"source"`
	Score          float64          `json:"score"`
	Variation      *string          `json:"variation"`
	StartDateTime  string           `json:"startDateTime"`
	EndDateTime    string           `json:"endDateTime"`
	Campaign       *Campaign        `json:"campaign"`
	Conditions     []LogicCondition `json:"conditions,omitempty"`
}

func (r ContentResult) ToResponse() ContentResult {
	return ContentResult{
		DecisionRuleId: r.DecisionRuleId,
		RuleSetType:    r.RuleSetType,
		ContentPath:    r.ContentPath,
		Source:         r.Source,
		Score:          r.Score,
		Variation:      r.Variation,
		StartDateTime:  r.StartDateTime,
		EndDateTime:    r.EndDateTime,
		Campaign:       r.Campaign,
	}
}

// func (r ContentResult) MarshalJSON() ([]byte, error) {
// 	type Alias ContentResult
// 	return json.Marshal(&struct {
// 		EvaluatedAt string `json:"evaluatedAt"`
// 		*Alias
// 	}{
// 		EvaluatedAt: r.EvaluatedAt,
// 		Alias:       (*Alias)(&r),
// 	})
// }

// LogicCondition is a flattened, self-contained representation of a single
// rule condition stored inside a ContentResult. It carries everything
// needed to evaluate the condition against live user attributes without a DB hit.
type LogicCondition struct {
	ConditionID       string          `json:"conditionId"`
	ParentConditionID string          `json:"parentConditionId,omitempty"` // empty = root
	AttributeID       string          `json:"attributeId"`
	DataType          string          `json:"dataType"`
	LogicalOperator   string          `json:"logicalOperator"`
	ConnectorOperator string          `json:"connectorOperator,omitempty"`
	Sequence          int             `json:"sequence"`
	ExpectedValue     json.RawMessage `json:"expectedValue"` // compact JSON from RuleAttribute.Value
}

// DeliveryService defines the cms-delivery contract.
// Responsible for serving pre-computed content results from Redis.
// This service has no PostgreSQL dependency; all reads come from the Redis cache.
type DeliveryService interface {
	// GetPersonalizedContent evaluates placement logic entries against the
	// caller's user attributes and returns ranked ContentResult items.
	// cisID identifies the Customer Information System channel; it is used to
	// scope the personalized placement cache key.
	// userAttrs maps attribute UUID strings to compact JSON values (the live
	// attribute values for this user, as received in the request payload).
	// Results are cached at cms:placement:{cisID}:{name} for resultTTL.
	GetPersonalizedContent(
		ctx context.Context,
		cisID, userID string,
		placementNames []string,
		userAttrs map[string]json.RawMessage,
	) ([]ContentResult, error)

	// FlushCache removes the cached results for the given placement names.
	// If placementNames is non-empty, only those placements are evicted.
	// If placementNames is nil or empty, ALL placement caches are flushed.
	FlushCache(ctx context.Context, placementNames []string, isEvaluate bool) error
}
