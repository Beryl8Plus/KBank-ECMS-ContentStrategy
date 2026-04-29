package repository

import (
	"context"

	"kbank-ecms/internal/domain/entity"
)

// CustomerProfileDataSource is one collection-projection request element.
type CustomerProfileDataSource struct {
	Datasource     string
	RequiredFields []string
}

// CustomerProfileQuery carries the inputs for a CLEN Customer Dynamic Query
// call. CisID is mandatory; DataSources lists the collections + fields to
// project from each.
type CustomerProfileQuery struct {
	CisID       string
	DataSources []CustomerProfileDataSource
}

// RawCustomerProfileResponse is the verbatim upstream payload used by the
// pass-through endpoint. StatusCode is the upstream HTTP status; Body is the
// raw JSON.
type RawCustomerProfileResponse struct {
	StatusCode int
	Body       []byte
}

// CustomerProfileRepository abstracts the upstream CLEN Customer Dynamic
// Query provider.
type CustomerProfileRepository interface {
	// GetCustomerProfile returns the decoded profile. A nil result with a nil
	// error means "no successful sources" — not an error.
	GetCustomerProfile(ctx context.Context, q CustomerProfileQuery) (*entity.CustomerProfile, error)

	// GetCustomerProfileRaw returns the upstream response verbatim (including
	// non-2xx status codes). Used by the /customer-profile pass-through
	// endpoint so the caller receives the same shape CLEN returned.
	GetCustomerProfileRaw(ctx context.Context, q CustomerProfileQuery) (*RawCustomerProfileResponse, error)
}
