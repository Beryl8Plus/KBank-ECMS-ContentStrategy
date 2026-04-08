package entity

import (
	"kbank-ecms/internal/domain/entity/enums"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// RuleCondition defines a single condition entry in the rule engine.
//
// Table: rule_conditions
type RuleCondition struct {
	BaseModel
	Sequence              int                     `gorm:"type:integer"                     json:"sequence"`
	DecisionRuleID        uuid.UUID               `gorm:"type:uuid;not null"               json:"decisionRuleId"`
	DecisionRule          *DecisionRule           `gorm:"foreignKey:DecisionRuleID"        json:"decisionRule,omitempty"`
	RuleID                *uuid.UUID              `gorm:"type:uuid"                        json:"ruleId"`
	Rule                  *Rule                   `gorm:"foreignKey:RuleID"                json:"rule,omitempty"`
	ParentRuleConditionID *uuid.UUID              `gorm:"type:uuid"                        json:"parentRuleConditionId"`
	ParentRuleCondition   *RuleCondition          `gorm:"foreignKey:ParentRuleConditionID" json:"parentRuleCondition,omitempty"`
	AttributeID           uuid.UUID               `gorm:"type:uuid"                        json:"attributeId"`
	Attribute             *Attribute              `gorm:"foreignKey:AttributeID"           json:"attribute,omitempty"`
	LogicalOperator       enums.LogicalOperator   `gorm:"size:255"                         json:"logicalOperator"`
	Value                 datatypes.JSON          `gorm:"type:jsonb"                       json:"value"`
	ConnectorOperator     enums.ConnectorOperator `gorm:"size:255"                         json:"connectorOperator"`
}

// TableName overrides default GORM table name to use plural form matching DBML.
func (RuleCondition) TableName() string {
	return "rule_conditions"
}
