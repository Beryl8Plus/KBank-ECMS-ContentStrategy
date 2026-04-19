package entity

import (
	"kbank-ecms/internal/domain/entity/enums"

	"github.com/google/uuid"
)

// RuleCondition defines a single condition entry in the rule engine.
//
// Table: rule_conditions
type RuleCondition struct {
	BaseModel
	Sequence              int                     `gorm:"type:integer"                                                                   json:"sequence"`
	DecisionRuleID        uuid.UUID               `gorm:"type:uuid;not null"                                                             json:"decisionRuleId"`
	DecisionRule          *DecisionRule           `gorm:"foreignKey:DecisionRuleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"        json:"decisionRule,omitempty"`
	ParentRuleConditionID *uuid.UUID              `gorm:"type:uuid"                                                                      json:"parentRuleConditionId"`
	ParentRuleCondition   *RuleCondition          `gorm:"foreignKey:ParentRuleConditionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"parentRuleCondition,omitempty"`
	AttributeID           uuid.UUID               `gorm:"type:uuid;not null"                                                             json:"attributeId"`
	Attribute             *Attribute              `gorm:"foreignKey:AttributeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"           json:"attribute,omitempty"`
	LogicalOperator       enums.LogicalOperator   `gorm:"size:255"                                                                       json:"logicalOperator"`
	ConnectorOperator     enums.ConnectorOperator `gorm:"size:255"                                                                       json:"connectorOperator"`
}

// TableName overrides default GORM table name to use plural form matching DBML.
func (RuleCondition) TableName() string {
	return "rule_conditions"
}
