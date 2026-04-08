package entity

import "github.com/google/uuid"

// ProfilePermission links a Profile to a Permission (many-to-many junction table).
//
// Table: profile_permissions (note: triple 's' preserved from DBML)
type ProfilePermission struct {
	BaseModel
	ProfileID    uuid.UUID   `gorm:"type:uuid;uniqueIndex:idx_profile_permission" json:"profileId"`
	Profile      *Profile    `gorm:"foreignKey:ProfileID"                         json:"profile,omitempty"`
	PermissionID uuid.UUID   `gorm:"type:uuid;uniqueIndex:idx_profile_permission" json:"permissionId"`
	Permission   *Permission `gorm:"foreignKey:PermissionID"                      json:"permission,omitempty"`
}

// TableName overrides default GORM table name to match the DBML (triple 's').
func (ProfilePermission) TableName() string {
	return "profile_permissions"
}
