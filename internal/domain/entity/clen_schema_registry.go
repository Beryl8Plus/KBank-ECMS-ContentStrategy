package entity

import "gorm.io/datatypes"

// CLENSchemaRegistry stores versioned JSON schemas for frontend display.
//
// Table: clen_schema_registry
type CLENSchemaRegistry struct {
	BaseModel
	SchemaName       string         `gorm:"size:255"      json:"schemaName"`
	Version          string         `gorm:"size:255"      json:"version"`
	SchemaDefinition datatypes.JSON `gorm:"type:jsonb"    json:"schemaDefinition"`
	IsActive         bool           `gorm:"default:false" json:"isActive"`
}

func (CLENSchemaRegistry) TableName() string {
	return "clen_schema_registry"
}
