package dto

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

// CreateAttributeRequest is the request body for POST /attributes.
type CreateAttributeRequest struct {
	ClenSchemaRegistryID uuid.UUID               `json:"clenSchemaRegistryId" binding:"required"`
	FieldName            string                  `json:"fieldName" binding:"required,max=255"`
	DisplayName          string                  `json:"displayName" binding:"required,max=255"`
	DataType             enums.AttributeDataType `json:"dataType" binding:"required"`
	Value                datatypes.JSON          `json:"value"`
	Description          string                  `json:"description"`
	SourceSystem         string                  `json:"sourceSystem" binding:"max=255"`
	IsActive             bool                    `json:"isActive"`
}

// UpdateAttributeRequest is the request body for PUT /attributes/:id.
type UpdateAttributeRequest struct {
	FieldName    string                  `json:"fieldName" binding:"required,max=255"`
	DisplayName  string                  `json:"displayName" binding:"required,max=255"`
	DataType     enums.AttributeDataType `json:"dataType" binding:"required"`
	Value        datatypes.JSON          `json:"value"`
	Description  string                  `json:"description"`
	SourceSystem string                  `json:"sourceSystem" binding:"max=255"`
	IsActive     bool                    `json:"isActive"`
}

// AttributeResponse is the response body for attribute endpoints.
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
	CreatedAt            time.Time               `json:"createdAt"`
	UpdatedAt            time.Time               `json:"updatedAt"`
}

// ToAttributeResponse converts an Attribute entity to an AttributeResponse DTO.
func ToAttributeResponse(a *entity.Attribute) AttributeResponse {
	return AttributeResponse{
		ID:                   a.ID,
		ClenSchemaRegistryID: a.ClenSchemaRegistryID,
		FieldName:            a.FieldName,
		DisplayName:          a.DisplayName,
		DataType:             a.DataType,
		Value:                a.Value,
		Description:          a.Description,
		SourceSystem:         a.SourceSystem,
		IsActive:             a.IsActive,
		CreatedAt:            a.CreatedAt,
		UpdatedAt:            a.UpdatedAt,
	}
}
