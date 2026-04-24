package entity

import (
	"kbank-ecms/internal/domain/entity/enums"

	"github.com/google/uuid"
)

// DecisionRule is the primary decision rule entity.
//
// Table: decision_rules
type DecisionRule struct {
	BaseModel
	DecisionRuleID string                      `gorm:"size:255;not null;uniqueIndex" json:"decisionRuleId"`
	Name           string                      `gorm:"size:255;not null"             json:"name"`
	Type           enums.DecisionType          `gorm:"size:255;not null"             json:"type"`
	EvaluateType   enums.EvaluateType          `gorm:"size:255;not null"             json:"evaluateType"`
	ContentPath    string                      `gorm:"size:255;not null"             json:"contentPath"`
	CampaignCode   string                      `gorm:"size:25;not null"              json:"campaignCode"`
	Score          float64                     `gorm:"type:float;default:0"          json:"score"`
	Status         enums.DecisionRuleStatus    `gorm:"size:255"                      json:"status"`
	SubStatus      enums.DecisionRuleSubStatus `gorm:"size:255"                      json:"subStatus"`
	InactiveBy     *uuid.UUID                  `gorm:"type:uuid"                     json:"inactiveBy"`
	InactiveByUser *User                       `gorm:"foreignKey:InactiveBy"         json:"inactiveByUser,omitempty"`
	// Associations used by the rule evaluation engine.
	RuleConditions []RuleCondition `gorm:"foreignKey:DecisionRuleID"     json:"ruleConditions,omitempty"`
	Rules          []Rule          `gorm:"foreignKey:DecisionRuleID"     json:"rules,omitempty"`
}
