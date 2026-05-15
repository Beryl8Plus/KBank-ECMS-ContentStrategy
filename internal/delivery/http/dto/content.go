package dto

import (
	"encoding/json"
	"math"

	"kbank-ecms/internal/domain/entity/enums"
)

// CustomerIdType identifies the scheme used for customerId.
type CustomerIdType string

const (
	CustomerIdTypeCISID             CustomerIdType = "CIS_ID"
	CustomerIdTypeIPID              CustomerIdType = "IP_ID"
	CustomerIdTypeKPlusMobileNumber CustomerIdType = "KPLUS_MOBILE_NUMBER"
	CustomerIdTypeLineUUID          CustomerIdType = "LINE_UUID"
)

type ContentRequestQueryParams struct {
	RequestType enums.RequestType `form:"requestType"         binding:"required,oneof=personalizedContent staticContent articleCategory"`
	Mode        string            `form:"mode"                binding:"required,oneof=knownContent logicalBased contentType articleCategory"`
	Channel     string            `form:"channel"             binding:"required"`
	Placements  []string          `form:"placement"           binding:"required,min=1,dive,required"`
	// CustomerID is required. When CustomerIDType is CIS_ID or IP_ID the value
	// must additionally be exactly 10 decimal digits (customer_id_format tag).
	CustomerID     string         `form:"customerId"          binding:"required,customer_id_format"`
	CustomerIDType CustomerIdType `form:"customerIdType"      binding:"required,oneof=CIS_ID IP_ID KPLUS_MOBILE_NUMBER LINE_UUID"`
	PageSize       int            `form:"pageSize,default=10" binding:"omitempty,max=2000"`
}

type CustomerRequest struct {
	Type              CustomerIdType `json:"customerIdType"`
	CIS_ID            string         `json:"cisId,omitempty"`
	IP_ID             string         `json:"ipId,omitempty"`
	KPlusMobileNumber string         `json:"kPlusMobileNumber,omitempty"`
	LineUUID          string         `json:"lineUuid,omitempty"`
}

func (c CustomerRequest) IsEmpty() bool {
	return c.CIS_ID == "" && c.IP_ID == "" && c.KPlusMobileNumber == "" && c.LineUUID == ""
}

func (c CustomerRequest) TypeName() string {
	return string(c.Type)
}

func (c CustomerRequest) Value() string {
	switch c.Type {
	case CustomerIdTypeCISID:
		return c.CIS_ID
	case CustomerIdTypeIPID:
		return c.IP_ID
	case CustomerIdTypeKPlusMobileNumber:
		return c.KPlusMobileNumber
	case CustomerIdTypeLineUUID:
		return c.LineUUID
	default:
		return ""
	}
}

// FlushRequest is the request body for POST /flush.
type FlushRequest struct {
	Placements []string `json:"placements"`
	IsEvaluate bool     `json:"isEvaluate"` // Optional flag to trigger cache re-population after flush
}

type FlushResponse struct {
	Message string `json:"message"`
}

// CacheStatusResponse is the response body for GET /purge_requests.
type CacheStatusResponse struct {
	IsMemPressure  bool     `json:"isMemPressure"`
	MemoryUsagePct float64  `json:"memoryUsagePct"`
	CacheKeys      []string `json:"cacheKeys"`
}

func (r CacheStatusResponse) MarshalJSON() ([]byte, error) {
	type Alias CacheStatusResponse
	// Ensure MemoryUsagePct is rounded to 4 decimal places for cleaner JSON output
	r.MemoryUsagePct = math.Round(r.MemoryUsagePct*10000) / 10000
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(&r),
	})
}

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
	ConditionID            string          `json:"conditionId"`
	ParentConditionID      string          `json:"parentConditionId,omitempty"` // empty = root
	AttributeID            string          `json:"attributeId"`
	DataType               string          `json:"dataType"`
	LogicalOperator        string          `json:"logicalOperator"`
	ConnectorOperator      string          `json:"connectorOperator,omitempty"`      // forward-link to next sibling
	ChildConnectorOperator string          `json:"childConnectorOperator,omitempty"` // joins own leaf check with children-group
	Sequence               int             `json:"sequence"`
	ExpectedValue          json.RawMessage `json:"expectedValue"` // compact JSON from RuleAttribute.Value
}

func ToContentResultResponses(results []ContentResult) []ContentResult {
	responses := make([]ContentResult, len(results))
	for i, r := range results {
		responses[i] = r.ToResponse()
	}
	return responses
}
