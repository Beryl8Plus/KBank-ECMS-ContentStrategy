package entity

import "github.com/google/uuid"

// Rule is a sub-rule / variation within a DecisionRule.
//
// Table: rules
type Rule struct {
	BaseModel
	DecisionRuleID uuid.UUID     `gorm:"type:uuid"                 json:"decisionRuleId"`
	DecisionRule   *DecisionRule `gorm:"foreignKey:DecisionRuleID" json:"decisionRule,omitempty"`
	VariationName  string        `gorm:"size:255"                  json:"variationName"`
	Score          int           `gorm:"type:integer"              json:"score"`
	OrderNo        int           `gorm:"type:integer"              json:"orderNo"`
}
