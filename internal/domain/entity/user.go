package entity

import "github.com/google/uuid"

// User represents a system user.
//
// Table: users
type User struct {
	BaseModel
	RoleID    *uuid.UUID `gorm:"type:uuid"               json:"roleId"`
	Role      *Role      `gorm:"foreignKey:RoleID"       json:"role,omitempty"`
	ProfileID *uuid.UUID `gorm:"type:uuid"               json:"profileId"`
	Profile   *Profile   `gorm:"foreignKey:ProfileID"    json:"profile,omitempty"`
	Email     string     `gorm:"size:255;uniqueIndex"    json:"email"`
	NameTH    string     `gorm:"size:255;column:name_th" json:"nameTh"`
	NameEN    string     `gorm:"size:255;column:name_en" json:"nameEn"`
	IsActive  bool       `gorm:"default:true"            json:"isActive"`
}
