package entity

// Placement is a master-data entity for content placement locations.
//
// Table: placements
type Placement struct {
	BaseModel
	Name        string `gorm:"size:255"   json:"name"`
	Description string `gorm:"type:text"  json:"description"`
	MaxResults  int    `gorm:"default:10" json:"maxResults"`
}
