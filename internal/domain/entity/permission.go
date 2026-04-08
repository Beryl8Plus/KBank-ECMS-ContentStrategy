package entity

import "kbank-ecms/internal/domain/entity/enums"

// Permission defines a granular access permission for a system resource.
//
// Table: permissions
type Permission struct {
	BaseModel
	Name           string                 `gorm:"size:255"                               json:"name"`   // Human-readable name for the permission, e.g. "Create User"
	PermissionType enums.PermissionType   `gorm:"size:255"                               json:"-"`      // Enum: ACCESS_CONTROL, FEATURE_FLAG, etc.
	Source         enums.Feature          `gorm:"size:255;uniqueIndex:idx_source_action" json:"source"` // Enum: CONTENT_DECISION_RULE, USER_MANAGEMENT, RULE_MANAGEMENT, etc. - identifies the resource or feature this permission applies to
	Action         enums.PermissionAction `gorm:"size:255;uniqueIndex:idx_source_action" json:"action"` // Enum: CREATE, EDIT, DELETE, VIEW_ALL, EDIT_ALL, DELETE_ALL, etc. - identifies the specific action allowed on the resource
}
