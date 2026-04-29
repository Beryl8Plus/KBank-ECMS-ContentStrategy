package repository

import (
	"context"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
)

// UserRepository defines the data-access contract for User lookups.
type UserRepository interface {
	// FindByIDs returns Users matching the given IDs (ID, NameTH, NameEN).
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error)
}
