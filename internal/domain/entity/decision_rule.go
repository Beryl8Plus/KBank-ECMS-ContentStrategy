package entity

import (
	"fmt"
	"kbank-ecms/internal/domain/entity/enums"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DecisionRule is the primary decision rule entity.
//
// Table: decision_rules
type DecisionRule struct {
	BaseModel
	// Human-readable running ID: RS-{YYYYMM}-{seq} e.g. RS-202503-0001
	DecisionRuleRunning string                      `gorm:"size:255;not null;uniqueIndex"                                           json:"decisionRuleId"`
	Name                string                      `gorm:"size:255;not null"                                                       json:"name"`
	Type                enums.DecisionType          `gorm:"size:255;not null"                                                       json:"type"`
	EvaluateType        enums.EvaluateType          `gorm:"size:255;not null"                                                       json:"evaluateType"`
	ContentPath         string                      `gorm:"size:255;not null"                                                       json:"contentPath"`
	CampaignCode        string                      `gorm:"size:25"                                                                 json:"campaignCode"`
	Score               float64                     `gorm:"type:float;default:0"                                                    json:"score"`
	Status              enums.DecisionRuleStatus    `gorm:"size:255"                                                                json:"status"`
	SubStatus           enums.DecisionRuleSubStatus `gorm:"size:255"                                                                json:"subStatus"`
	InactiveBy          *uuid.UUID                  `gorm:"type:uuid"                                                               json:"inactiveBy"`
	InactiveByUser      *User                       `gorm:"foreignKey:InactiveBy"                                                   json:"inactiveByUser,omitempty"`
	// Associations used by the rule evaluation engine.
	RuleConditions []RuleCondition `gorm:"foreignKey:DecisionRuleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"ruleConditions,omitempty"`
	Rules          []Rule          `gorm:"foreignKey:DecisionRuleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"rules,omitempty"`
}

// BeforeCreate auto-generates DecisionRuleID in RS-{YYYYMM}-{seq} format when not already set.
func (d *DecisionRule) BeforeCreate(tx *gorm.DB) error {
	if err := d.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}
	if d.DecisionRuleRunning != "" {
		return nil
	}
	var id string
	if err := tx.Raw("SELECT next_decision_rule_id()").Scan(&id).Error; err != nil {
		return fmt.Errorf("generating decision rule ID: %w", err)
	}
	d.DecisionRuleRunning = id
	tx.Statement.SetColumn("DecisionRuleRunning", id)
	return nil
}
