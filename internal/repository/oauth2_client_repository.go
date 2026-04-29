package repository

import (
	"context"
	"fmt"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"

	"gorm.io/gorm"
)

// OAuth2ClientPostgresRepository implements OAuth2ClientRepository using GORM.
type OAuth2ClientPostgresRepository struct {
	db *gorm.DB
}

var _ domainrepo.OAuth2ClientRepository = (*OAuth2ClientPostgresRepository)(nil)

// NewOAuth2ClientPostgresRepository creates a new repository instance.
func NewOAuth2ClientPostgresRepository(db *gorm.DB) *OAuth2ClientPostgresRepository {
	return &OAuth2ClientPostgresRepository{db: db}
}

// GetByClientID returns the OAuth2 client matching the given client_id.
func (r *OAuth2ClientPostgresRepository) GetByClientID(ctx context.Context, clientID string) (*entity.OAuth2Client, error) {
	var client entity.OAuth2Client
	err := r.db.WithContext(ctx).
		Where(`"CLIENT_ID" = ? AND "IS_ACTIVE" = TRUE AND "DELETED_AT" IS NULL`, clientID).
		First(&client).Error
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// GetClientScopes returns the scopes (formatted as "source:action") granted
// to the client based on its profile's permissions.
func (r *OAuth2ClientPostgresRepository) GetClientScopes(ctx context.Context, clientID string) ([]string, error) {
	type scopeRow struct {
		Source string
		Action string
	}

	var rows []scopeRow
	err := r.db.WithContext(ctx).
		Table(`oauth2_clients AS oc`).
		Select(`p."SOURCE" AS source, p."ACTION" AS action`).
		Joins(`INNER JOIN profile_permissions AS pp ON pp."PROFILE_ID" = oc."PROFILE_ID" AND pp."DELETED_AT" IS NULL`).
		Joins(`INNER JOIN permissions AS p ON p."ID" = pp."PERMISSION_ID" AND p."DELETED_AT" IS NULL`).
		Where(`oc."CLIENT_ID" = ? AND oc."IS_ACTIVE" = TRUE AND oc."DELETED_AT" IS NULL`, clientID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	scopes := make([]string, 0, len(rows))
	for _, row := range rows {
		scopes = append(scopes, fmt.Sprintf("%s:%s", row.Source, row.Action))
	}
	return scopes, nil
}
