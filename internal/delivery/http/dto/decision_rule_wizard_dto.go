package dto

import (
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

// ── Step 1 ───────────────────────────────────────────────────────────────────

// WizardStep1Request is the body for POST /decision-rules.
type WizardStep1Request struct {
	Type         enums.DecisionType     `json:"type"         binding:"required"`
	EvaluateType enums.EvaluateType     `json:"evaluateType" binding:"required"`
	Name         string                 `json:"name"         binding:"required,max=255"`
	ContentPath  string                 `json:"contentPath"  binding:"required,max=255"`
	CampaignCode string                 `json:"campaignCode" binding:"max=25"`
	Score        float64                `json:"score"        binding:"min=0,max=100"`
	Conditions   []ConditionItemRequest `json:"conditions"  binding:"required,min=1"`
}

// ConditionItemRequest is a recursive condition node (type = condition | group).
type ConditionItemRequest struct {
	Type              enums.ConditionType     `json:"type"              binding:"required"`
	Sequence          int                     `json:"sequence"          binding:"required,min=1"`
	AttributeID       uuid.UUID               `json:"attributeId"`
	LogicalOperator   enums.LogicalOperator   `json:"logicalOperator"`
	ConnectorOperator enums.ConnectorOperator `json:"connectorOperator"`
	Conditions        []ConditionItemRequest  `json:"conditions"`
}

// WizardStep1Response is returned after a successful Step 1 save.
type WizardStep1Response struct {
	ID                  uuid.UUID                `json:"id"`
	DecisionRuleRunning string                   `json:"decisionRuleId"`
	Status              enums.DecisionRuleStatus `json:"status"`
	CreatedAt           time.Time                `json:"createdAt"`
}

// ── Step 1 read (edit mode) ───────────────────────────────────────────────────

// WizardConditionsResponse is returned by GET /decision-rules/:id/conditions.
type WizardConditionsResponse struct {
	ID                  uuid.UUID                   `json:"id"`
	DecisionRuleRunning string                      `json:"decisionRuleId"`
	Type                enums.DecisionType          `json:"type"`
	EvaluateType        enums.EvaluateType          `json:"evaluateType"`
	Name                string                      `json:"name"`
	ContentPath         string                      `json:"contentPath"`
	CampaignCode        string                      `json:"campaignCode"`
	Score               float64                     `json:"score"`
	Status              enums.DecisionRuleStatus    `json:"status"`
	SubStatus           enums.DecisionRuleSubStatus `json:"subStatus"`
	Conditions          []ConditionItemResponse     `json:"conditions"`
}

// ConditionItemResponse is a recursive condition node returned by the GET endpoints.
type ConditionItemResponse struct {
	ConditionID       uuid.UUID               `json:"conditionId"`
	Type              enums.ConditionType     `json:"type"`
	Sequence          int                     `json:"sequence"`
	AttributeID       *uuid.UUID              `json:"attributeId,omitempty"`
	AttributeName     string                  `json:"attributeName,omitempty"`
	AttributeIsActive bool                    `json:"attributeIsActive"`
	LogicalOperator   enums.LogicalOperator   `json:"logicalOperator,omitempty"`
	ConnectorOperator enums.ConnectorOperator `json:"connectorOperator"`
	Conditions        []ConditionItemResponse `json:"conditions,omitempty"`
}

// ── Step 2 ───────────────────────────────────────────────────────────────────

// WizardRuleSetsResponse is returned by GET /decision-rules/:id/rule-sets.
type WizardRuleSetsResponse struct {
	ID       uuid.UUID            `json:"id"`
	Columns  []RuleColumnResponse `json:"columns"`
	RuleSets []RuleSetResponse    `json:"ruleSets"`
}

// RuleColumnResponse describes one column header in the rule table.
type RuleColumnResponse struct {
	ConditionID       uuid.UUID               `json:"conditionId"`
	AttributeID       uuid.UUID               `json:"attributeId"`
	AttributeName     string                  `json:"attributeName"`
	AttributeIsActive bool                    `json:"attributeIsActive"`
	LogicalOperator   enums.LogicalOperator   `json:"logicalOperator"`
	DataType          enums.AttributeDataType `json:"dataType"`
}

// RuleSetResponse is one row in the rule table.
type RuleSetResponse struct {
	RuleID        uuid.UUID           `json:"ruleId"`
	OrderNo       int                 `json:"orderNo"`
	Score         *float64            `json:"score"`
	VariationName string              `json:"variationName"`
	Values        []RuleValueResponse `json:"values"`
}

// RuleValueResponse is a (conditionId, value) cell in the rule table.
type RuleValueResponse struct {
	ConditionID uuid.UUID `json:"conditionId"`
	Value       *string   `json:"value"`
}

// WizardStep2Request is the body for PUT /decision-rules/:id/rule-sets.
type WizardStep2Request struct {
	RuleSets []RuleSetUpsertRequest `json:"ruleSets" binding:"required,min=1"`
}

// RuleSetUpsertRequest is one rule row to create or update.
type RuleSetUpsertRequest struct {
	RuleID        *uuid.UUID              `json:"ruleId"`
	OrderNo       int                     `json:"orderNo"        binding:"required,min=1"`
	Score         *float64                `json:"score"`
	VariationName string                  `json:"variationName"  binding:"required,max=255"`
	Conditions    []ConditionValueRequest `json:"conditions"     binding:"required"`
}

// ConditionValueRequest links a conditionId (from Step 1) to a value.
type ConditionValueRequest struct {
	ConditionID uuid.UUID `json:"conditionId" binding:"required"`
	Value       *string   `json:"value"`
}

// WizardStep2Response is returned after a successful Step 2 save.
type WizardStep2Response struct {
	ID            uuid.UUID `json:"id"`
	SavedRuleSets int       `json:"savedRuleSets"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ── Step 3 ───────────────────────────────────────────────────────────────────

// WizardStep3Request is the body for PUT /decision-rules/:id/schedules.
type WizardStep3Request struct {
	Schedules []ScheduleUpsertRequest `json:"schedules" binding:"required,min=1"`
}

// ScheduleUpsertRequest is one schedule to create.
type ScheduleUpsertRequest struct {
	PlacementID    uuid.UUID            `json:"placementId"    binding:"required"`
	StartDate      time.Time            `json:"startDate"      binding:"required"`
	EndDate        time.Time            `json:"endDate"        binding:"required"`
	RecurrenceType enums.RecurrenceType `json:"recurrenceType" binding:"required"`
	RecurrenceRule *string              `json:"recurrenceRule"`
	CalendarID     *uuid.UUID           `json:"calendarId"`
	AllDay         bool                 `json:"allDay"`
	TimeOfDayStart *string              `json:"timeOfDayStart"`
	TimeOfDayEnd   *string              `json:"timeOfDayEnd"`
	Timezone       string               `json:"timezone"`
}

// WizardStep3Response is returned after a successful Step 3 save.
type WizardStep3Response struct {
	ID                  uuid.UUID                `json:"id"`
	DecisionRuleRunning string                   `json:"decisionRuleId"`
	Status              enums.DecisionRuleStatus `json:"status"`
	Schedules           []WizardScheduleResponse `json:"schedules"`
}

// WizardStep4Response is returned after a successful Step 4 activate.
type WizardStep4Response struct {
	ID                  uuid.UUID                `json:"id"`
	DecisionRuleRunning string                   `json:"decisionRuleId"`
	Status              enums.DecisionRuleStatus `json:"status"`
	Schedules           []*entity.Schedule
}

// WizardScheduleResponse describes one saved schedule with placement details.
type WizardScheduleResponse struct {
	ScheduleID     uuid.UUID            `json:"scheduleId"`
	PlacementID    uuid.UUID            `json:"placementId"`
	PlacementName  string               `json:"placementName"`
	ChannelID      uuid.UUID            `json:"channelId"`
	ChannelName    string               `json:"channelName"`
	StartDate      time.Time            `json:"startDate"`
	EndDate        time.Time            `json:"endDate"`
	RecurrenceType enums.RecurrenceType `json:"recurrenceType"`
	AllDay         bool                 `json:"allDay"`
	Timezone       string               `json:"timezone"`
	IsActive       bool                 `json:"isActive"`
}

// WizardSchedulesResponse is returned by GET /decision-rules/:id/schedules.
type WizardSchedulesResponse struct {
	ID                  uuid.UUID                `json:"id"`
	DecisionRuleRunning string                   `json:"decisionRuleId"`
	Schedules           []WizardScheduleResponse `json:"schedules"`
}

// ── List ─────────────────────────────────────────────────────────────────────

// DecisionRulePlacementResponse is one placement entry in the list response.
type DecisionRulePlacementResponse struct {
	PlacementID   uuid.UUID `json:"placementId"`
	PlacementName string    `json:"placementName"`
	ChannelName   string    `json:"channelName"`
}

// DecisionRuleListItemResponse is one row in GET /decision-rules.
type DecisionRuleListItemResponse struct {
	ID                  uuid.UUID                       `json:"id"`
	DecisionRuleRunning string                          `json:"decisionRuleId"`
	Name                string                          `json:"name"`
	Type                enums.DecisionType              `json:"type"`
	EvaluateType        enums.EvaluateType              `json:"evaluateType"`
	CampaignCode        string                          `json:"campaignCode"`
	Status              enums.DecisionRuleStatus        `json:"status"`
	SubStatus           enums.DecisionRuleSubStatus     `json:"subStatus"`
	Placements          []DecisionRulePlacementResponse `json:"placements"`
	CreatedAt           time.Time                       `json:"createdAt"`
	UpdatedAt           time.Time                       `json:"updatedAt"`
}

// ── Mapper helpers ────────────────────────────────────────────────────────────

// ToWizardScheduleResponse converts a Schedule entity (with Placement+Channel preloaded)
// to a WizardScheduleResponse.
func ToWizardScheduleResponse(s *entity.Schedule) WizardScheduleResponse {
	r := WizardScheduleResponse{
		ScheduleID:     s.ID,
		PlacementID:    s.PlacementID,
		StartDate:      s.EffectiveFrom,
		EndDate:        s.EffectiveUntil,
		RecurrenceType: s.RecurrenceType,
		AllDay:         s.AllDay,
		Timezone:       s.Timezone,
		IsActive:       s.IsActive,
	}
	if s.Placement != nil {
		r.PlacementName = s.Placement.PlacementName
		if s.Placement.Channel != nil {
			r.ChannelID = s.Placement.Channel.ID
			r.ChannelName = s.Placement.Channel.ChannelName
		}
	}
	return r
}
