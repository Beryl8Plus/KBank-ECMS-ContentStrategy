package entity

import (
	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity/enums"
)

// RuleCondition defines a single condition entry in the decision rule engine.
// Rows with AttributeID = nil are group containers (type = "group").
//
// Table: rule_conditions
type RuleCondition struct {
	BaseModel
	Sequence               int                      `gorm:"type:integer"                                                                   json:"sequence"`
	DecisionRuleID         uuid.UUID                `gorm:"type:uuid;not null"                                                             json:"decisionRuleId"`
	DecisionRule           *DecisionRule            `gorm:"foreignKey:DecisionRuleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"        json:"decisionRule,omitempty"`
	ParentRuleConditionID  *uuid.UUID               `gorm:"type:uuid"                                                                      json:"parentRuleConditionId"`
	ParentRuleCondition    *RuleCondition           `gorm:"foreignKey:ParentRuleConditionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"parentRuleCondition,omitempty"`
	RuleConditionChildren  []RuleCondition          `gorm:"foreignKey:ParentRuleConditionID;references:ID"                                 json:"ruleConditionChildren,omitempty"`
	AttributeID            *uuid.UUID               `gorm:"type:uuid"                                                                      json:"attributeId"`
	Attribute              *Attribute               `gorm:"foreignKey:AttributeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"           json:"attribute,omitempty"`
	LogicalOperator        enums.LogicalOperator    `gorm:"size:50"                                                                        json:"logicalOperator"`
	ConnectorOperator      *enums.ConnectorOperator `gorm:"size:50"                                                                        json:"connectorOperator"`
	ChildConnectorOperator *enums.ConnectorOperator `gorm:"size:50"                                                                        json:"childConnectorOperator"`
}

// TableName overrides default GORM table name to use plural form matching DBML.
func (RuleCondition) TableName() string {
	return "rule_conditions"
}
