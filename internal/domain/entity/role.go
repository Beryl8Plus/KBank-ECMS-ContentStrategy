package entity

// Role defines a user role within the system.
//
// Table: roles
type Role struct {
	BaseModel
	Name string `gorm:"size:255"             json:"name"`
	Code string `gorm:"size:255;uniqueIndex" json:"code"`
}
