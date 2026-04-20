package dto

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

// AttributeResponse represents the core attribute fields.
type AttributeResponse struct {
	ID                   uuid.UUID               `json:"id"`
	ClenSchemaRegistryID uuid.UUID               `json:"clenSchemaRegistryId"`
	FieldName            string                  `json:"fieldName"`
	DisplayName          string                  `json:"displayName"`
	DataType             enums.AttributeDataType `json:"dataType"`
	Value                datatypes.JSON          `json:"value"`
	Description          string                  `json:"description"`
	SourceSystem         string                  `json:"sourceSystem"`
	IsActive             bool                    `json:"isActive"`
}

// RuleConditionResponse maps the RuleCondition entity.
type RuleConditionResponse struct {
	ID                    uuid.UUID               `json:"id"`
	Sequence              int                     `json:"sequence"`
	DecisionRuleID        uuid.UUID               `json:"decisionRuleId"`
	ParentRuleConditionID *uuid.UUID              `json:"parentRuleConditionId,omitempty"`
	AttributeID           uuid.UUID               `json:"attributeId"`
	Attribute             *AttributeResponse      `json:"attribute,omitempty"`
	LogicalOperator       enums.LogicalOperator   `json:"logicalOperator"`
	ConnectorOperator     enums.ConnectorOperator `json:"connectorOperator"`
	CreatedAt             time.Time               `json:"createdAt"`
	UpdatedAt             time.Time               `json:"updatedAt"`
}

// RuleAttributeResponse maps the RuleAttribute entity.
type RuleAttributeResponse struct {
	ID          uuid.UUID          `json:"id"`
	RuleID      uuid.UUID          `json:"ruleId"`
	AttributeID uuid.UUID          `json:"attributeId"`
	Attribute   *AttributeResponse `json:"attribute,omitempty"`
	Value       datatypes.JSON     `json:"value"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}

// RuleResponse maps the Rule entity.
type RuleResponse struct {
	ID             uuid.UUID               `json:"id"`
	DecisionRuleID uuid.UUID               `json:"decisionRuleId"`
	VariationName  string                  `json:"variationName"`
	Score          float64                 `json:"score"`
	OrderNo        int                     `json:"orderNo"`
	RuleAttributes []RuleAttributeResponse `json:"ruleAttributes,omitempty"`
	CreatedAt      time.Time               `json:"createdAt"`
	UpdatedAt      time.Time               `json:"updatedAt"`
}

// DecisionRuleResponse maps the DecisionRule entity and its relationships.
type DecisionRuleResponse struct {
	ID             uuid.UUID                `json:"id"`
	Name           string                   `json:"name"`
	Type           enums.DecisionType       `json:"type"`
	EvaluateType   enums.EvaluateType       `json:"evaluateType"`
	ContentPath    string                   `json:"contentPath"`
	Score          float64                  `json:"score"`
	Status         enums.DecisionRuleStatus `json:"status"`
	InactiveBy     *uuid.UUID               `json:"inactiveBy,omitempty"`
	RuleConditions []RuleConditionResponse  `json:"ruleConditions,omitempty"`
	Rules          []RuleResponse           `json:"rules,omitempty"`
	CreatedAt      time.Time                `json:"createdAt"`
	UpdatedAt      time.Time                `json:"updatedAt"`
}

func ToAttributeResponse(a *entity.Attribute) *AttributeResponse {
	if a == nil {
		return nil
	}
	return &AttributeResponse{
		ID:                   a.ID,
		ClenSchemaRegistryID: a.ClenSchemaRegistryID,
		FieldName:            a.FieldName,
		DisplayName:          a.DisplayName,
		DataType:             a.DataType,
		Value:                a.Value,
		Description:          a.Description,
		SourceSystem:         a.SourceSystem,
		IsActive:             a.IsActive,
	}
}

// ToDecisionRuleResponse maps a complete DecisionRule to a DecisionRuleResponse DTO.
func ToDecisionRuleResponse(dr *entity.DecisionRule) DecisionRuleResponse {
	resp := DecisionRuleResponse{
		ID:           dr.ID,
		Name:         dr.Name,
		Type:         dr.Type,
		EvaluateType: dr.EvaluateType,
		ContentPath:  dr.ContentPath,
		Score:        dr.Score,
		Status:       dr.Status,
		InactiveBy:   dr.InactiveBy,
		CreatedAt:    dr.CreatedAt,
		UpdatedAt:    dr.UpdatedAt,
	}

	if len(dr.RuleConditions) > 0 {
		rcResp := make([]RuleConditionResponse, len(dr.RuleConditions))
		for i, rc := range dr.RuleConditions {
			rcResp[i] = RuleConditionResponse{
				ID:                    rc.ID,
				Sequence:              rc.Sequence,
				DecisionRuleID:        rc.DecisionRuleID,
				ParentRuleConditionID: rc.ParentRuleConditionID,
				AttributeID:           rc.AttributeID,
				Attribute:             ToAttributeResponse(rc.Attribute),
				LogicalOperator:       rc.LogicalOperator,
				ConnectorOperator:     rc.ConnectorOperator,
				CreatedAt:             rc.CreatedAt,
				UpdatedAt:             rc.UpdatedAt,
			}
		}
		resp.RuleConditions = rcResp
	}

	if len(dr.Rules) > 0 {
		rResp := make([]RuleResponse, len(dr.Rules))
		for i, r := range dr.Rules {
			rResp[i] = RuleResponse{
				ID:             r.ID,
				DecisionRuleID: r.DecisionRuleID,
				VariationName:  r.VariationName,
				Score:          r.Score,
				OrderNo:        r.OrderNo,
				CreatedAt:      r.CreatedAt,
				UpdatedAt:      r.UpdatedAt,
			}

			if len(r.RuleAttributes) > 0 {
				raResp := make([]RuleAttributeResponse, len(r.RuleAttributes))
				for j, ra := range r.RuleAttributes {
					raResp[j] = RuleAttributeResponse{
						ID:          ra.ID,
						RuleID:      ra.RuleID,
						AttributeID: ra.AttributeID,
						Attribute:   ToAttributeResponse(ra.Attribute),
						Value:       ra.Value,
						CreatedAt:   ra.CreatedAt,
						UpdatedAt:   ra.UpdatedAt,
					}
				}
				rResp[i].RuleAttributes = raResp
			}
		}
		resp.Rules = rResp
	}

	return resp
}
