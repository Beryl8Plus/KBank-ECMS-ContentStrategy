package ctxconsts

import (
	"context"

	"gorm.io/gorm"
)

// contextKey is an unexported type for context keys in this package.
// Using a custom type prevents collision with keys from other packages.
type contextKey string

const (
	// UserIDKey is the context key for the current user's UUID
	UserIDKey contextKey = "userID"

	// DBKey is the context key for the GORM database instance
	DBKey contextKey = "DB"
)

// GetUserID retrieves the user ID from the context.
// It returns the user ID and a boolean indicating if it was found.
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// GetDB retrieves the GORM DB instance from the context.
// It returns the *gorm.DB instance and a boolean indicating if it was found.
func GetDB(ctx context.Context) (*gorm.DB, bool) {
	db, ok := ctx.Value(DBKey).(*gorm.DB)
	return db, ok
}
