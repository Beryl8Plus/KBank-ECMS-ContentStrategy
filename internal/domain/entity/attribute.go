package entity

import (
	"kbank-ecms/internal/domain/entity/enums"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Attribute defines a data attribute used in rule conditions.
//
// Table: attributes
type Attribute struct {
	BaseModel
	MdpSchemaRegistryID uuid.UUID               `gorm:"type:uuid;not null" json:"mdpSchemaRegistryId"`
	FieldName           string                  `gorm:"size:255"           json:"fieldName"`
	DisplayName         string                  `gorm:"size:255"           json:"displayName"`
	DataType            enums.AttributeDataType `gorm:"size:255"           json:"dataType"`
	Value               datatypes.JSON          `gorm:"type:jsonb"         json:"value"`
	Description         string                  `gorm:"type:text"          json:"description"`
	SourceSystem        string                  `gorm:"size:255"           json:"sourceSystem"`
	IsActive            bool                    `gorm:"default:true"       json:"isActive"`
}
