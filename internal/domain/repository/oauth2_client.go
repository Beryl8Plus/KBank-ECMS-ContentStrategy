package repository

import (
	"context"

	"kbank-ecms/internal/domain/entity"
)

// OAuth2ClientRepository defines the contract for accessing OAuth2 clients.
type OAuth2ClientRepository interface {
	// GetByClientID returns the OAuth2 client matching the given client_id.
	GetByClientID(ctx context.Context, clientID string) (*entity.OAuth2Client, error)

	// GetClientScopes returns the scopes (formatted as "source:action") granted
	// to the client based on its profile's permissions.
	GetClientScopes(ctx context.Context, clientID string) ([]string, error)
}
