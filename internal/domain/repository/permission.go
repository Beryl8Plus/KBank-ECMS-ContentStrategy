package repository

import (
	"context"

	"github.com/google/uuid"
)

// PermissionRepository defines the contract for checking profile-based permissions.
type PermissionRepository interface {
	// HasPermission reports whether the user's active profile grants the given source+action.
	HasPermission(ctx context.Context, userID uuid.UUID, source, action string) (bool, error)
}
