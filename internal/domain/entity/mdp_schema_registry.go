package entity

import "gorm.io/datatypes"

// SchemaRegistry stores versioned JSON schemas for frontend display.
//
// Table: schema_registry
type MDPSchemaRegistry struct {
	BaseModel
	SchemaName       string         `gorm:"size:255"      json:"schemaName"`
	Version          string         `gorm:"size:255"      json:"version"`
	SchemaDefinition datatypes.JSON `gorm:"type:jsonb"    json:"schemaDefinition"`
	IsActive         bool           `gorm:"default:false" json:"isActive"`
}

func (MDPSchemaRegistry) TableName() string {
	return "mdp_schema_registry"
}
