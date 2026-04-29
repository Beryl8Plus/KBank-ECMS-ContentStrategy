package entity

import (
	"encoding/json"
	"fmt"
	"kbank-ecms/internal/domain/entity/enums"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Attribute defines a data attribute used in rule conditions.
//
// Table: attributes
type Attribute struct {
	BaseModel
	ClenSchemaRegistryID uuid.UUID               `gorm:"type:uuid;not null" json:"clenSchemaRegistryId"`
	FieldName            string                  `gorm:"size:255"           json:"fieldName"`
	DisplayName          string                  `gorm:"size:255"           json:"displayName"`
	DataType             enums.AttributeDataType `gorm:"size:255"           json:"dataType"`
	Value                datatypes.JSON          `gorm:"type:jsonb"         json:"value"`
	Description          string                  `gorm:"type:text"          json:"description"`
	SourceSystem         string                  `gorm:"size:255"           json:"sourceSystem"`
	TableSourceName      string                  `gorm:"size:255"           json:"tableSourceName"`
	IsActive             bool                    `gorm:"default:true"       json:"isActive"`
}

// ValidOptions parses Value jsonb into a set of allowed strings.
// Returns nil when Value is empty (meaning no constraint — any value is accepted).
// Supports two JSON formats:
//   - Simple list:      ["GOLD", "SILVER"]
//   - Structured list:  [{"value":"GOLD"}, {"value":"SILVER"}]
func (a *Attribute) ValidOptions() (map[string]struct{}, error) {
	if len(a.Value) == 0 {
		return nil, nil
	}

	var simple []string
	if err := json.Unmarshal(a.Value, &simple); err == nil {
		set := make(map[string]struct{}, len(simple))
		for _, v := range simple {
			set[v] = struct{}{}
		}
		return set, nil
	}

	var structured []struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(a.Value, &structured); err != nil {
		return nil, fmt.Errorf("attribute %s: cannot parse value options: %w", a.ID, err)
	}
	set := make(map[string]struct{}, len(structured))
	for _, v := range structured {
		set[v.Value] = struct{}{}
	}
	return set, nil
}
