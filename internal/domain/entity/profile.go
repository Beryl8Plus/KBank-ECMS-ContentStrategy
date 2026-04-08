package entity

// Profile defines a user profile group.
//
// Table: profiles
type Profile struct {
	BaseModel
	Name string `gorm:"size:255"             json:"name"`
	Code string `gorm:"size:255;uniqueIndex" json:"code"`
}
