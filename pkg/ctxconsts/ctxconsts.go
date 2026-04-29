package ctxconsts

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// contextKey is an unexported type for context keys in this package.
// Using a custom type prevents collision with keys from other packages.
type contextKey string

const (
	// UserIDKey is the context key for the current user's UUID
	UserIDKey contextKey = "userID"
	// CisIDKey is the context key for the current user's CIS ID
	CisIDKey contextKey = "cisID"

	// DBKey is the context key for the GORM database instance
	DBKey contextKey = "DB"

	// CorrelationIDKey is the context key for the inbound request correlation ID (requestID header).
	CorrelationIDKey contextKey = "correlationID"

	// ClientIDKey is the context key for the OAuth2 client_id when the request was
	// authenticated via Client Credentials Flow.
	ClientIDKey contextKey = "clientID"

	// ScopesKey is the context key for the OAuth2 scopes granted to the current request.
	ScopesKey contextKey = "scopes"
)

// GetClientID retrieves the OAuth2 client_id from the context.
func GetClientID(ctx context.Context) (string, bool) {
	clientID, ok := ctx.Value(ClientIDKey).(string)
	return clientID, ok
}

// GetScopes retrieves the OAuth2 scopes granted to the current request from the context.
func GetScopes(ctx context.Context) ([]string, bool) {
	scopes, ok := ctx.Value(ScopesKey).([]string)
	return scopes, ok
}

// GetUserID retrieves the user ID from the context.
// It returns the user ID string and a boolean indicating if it was found.
func GetUserID(ctx context.Context) (*uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		return nil, false
	}
	return &userID, true
}

// GetCisID retrieves the CIS ID from the context.
// It returns the CIS ID and a boolean indicating if it was found.
func GetCisID(ctx context.Context) (string, bool) {
	cisID, ok := ctx.Value(CisIDKey).(string)
	return cisID, ok
}

// GetDB retrieves the GORM DB instance from the context.
// It returns the *gorm.DB instance and a boolean indicating if it was found.
func GetDB(ctx context.Context) (*gorm.DB, bool) {
	db, ok := ctx.Value(DBKey).(*gorm.DB)
	return db, ok
}

// SetCorrelationID returns a new context with the given correlation ID stored under CorrelationIDKey.
func SetCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, id)
}

// GetCorrelationID retrieves the correlation ID from the context.
// Returns an empty string when not set.
func GetCorrelationID(ctx context.Context) string {
	id, _ := ctx.Value(CorrelationIDKey).(string)
	return id
}
