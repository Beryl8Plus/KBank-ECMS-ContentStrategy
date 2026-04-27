package entity

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// RuleAttribute stores the value a specific Rule assigns to one attribute column.
//
// Table: rule_attributes
type RuleAttribute struct {
	BaseModel
	RuleID      uuid.UUID      `gorm:"type:uuid;not null"                                                   json:"ruleId"`
	Rule        *Rule          `gorm:"foreignKey:RuleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"      json:"rule,omitempty"`
	AttributeID uuid.UUID      `gorm:"type:uuid;not null"                                                   json:"attributeId"`
	Attribute   *Attribute     `gorm:"foreignKey:AttributeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"attribute,omitempty"`
	Value       datatypes.JSON `gorm:"type:jsonb"                                                           json:"value"`
}

func (RuleAttribute) TableName() string {
	return "rule_attributes"
}
