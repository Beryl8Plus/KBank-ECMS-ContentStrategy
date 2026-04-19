package entity

import "github.com/google/uuid"

// Rule is a sub-rule / variation within a DecisionRule.
//
// Table: rules
type Rule struct {
	BaseModel
	DecisionRuleID uuid.UUID       `gorm:"type:uuid;not null"                                                      json:"decisionRuleId"`
	DecisionRule   *DecisionRule   `gorm:"foreignKey:DecisionRuleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"decisionRule,omitempty"`
	VariationName  string          `gorm:"size:255"                                                                json:"variationName"`
	Score          float64         `gorm:"type:float"                                                              json:"score"`
	OrderNo        int             `gorm:"type:integer"                                                            json:"orderNo"`
	RuleAttributes []RuleAttribute `gorm:"foreignKey:RuleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"         json:"ruleAttributes,omitempty"`
}

func (Rule) TableName() string {
	return "rules"
}
