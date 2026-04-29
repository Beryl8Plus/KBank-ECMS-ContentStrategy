package repository

import (
	"context"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	httpclient "kbank-ecms/internal/http/client"
)

// CLENCustomerProfileRepository adapts *client.CLENCustomerProfileClient to
// the domain CustomerProfileRepository interface.
type CLENCustomerProfileRepository struct {
	client *httpclient.CLENCustomerProfileClient
}

// Compile-time interface check.
var _ domainrepo.CustomerProfileRepository = (*CLENCustomerProfileRepository)(nil)

// NewCLENCustomerProfileRepository accepts a nil client (feature-off). In
// that case both methods return empty/nil values without error, letting the
// delivery service proceed without profile data.
func NewCLENCustomerProfileRepository(c *httpclient.CLENCustomerProfileClient) *CLENCustomerProfileRepository {
	return &CLENCustomerProfileRepository{client: c}
}

func (r *CLENCustomerProfileRepository) GetCustomerProfile(ctx context.Context, q domainrepo.CustomerProfileQuery) (*entity.CustomerProfile, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	return r.client.GetCustomerProfile(ctx, toClientRequest(q))
}

func (r *CLENCustomerProfileRepository) GetCustomerProfileRaw(ctx context.Context, q domainrepo.CustomerProfileQuery) (*domainrepo.RawCustomerProfileResponse, error) {
	if r == nil || r.client == nil {
		return &domainrepo.RawCustomerProfileResponse{StatusCode: 200, Body: []byte("{}")}, nil
	}
	raw, err := r.client.GetCustomerProfileRaw(ctx, toClientRequest(q))
	if err != nil {
		return nil, err
	}
	return &domainrepo.RawCustomerProfileResponse{StatusCode: raw.StatusCode, Body: raw.Body}, nil
}

func toClientRequest(q domainrepo.CustomerProfileQuery) httpclient.CustomerProfileQueryRequest {
	sources := make([]httpclient.CustomerProfileDataSourceReq, 0, len(q.DataSources))
	for _, ds := range q.DataSources {
		sources = append(sources, httpclient.CustomerProfileDataSourceReq{
			Datasource:     ds.Datasource,
			RequiredFields: ds.RequiredFields,
		})
	}
	return httpclient.CustomerProfileQueryRequest{
		CisID:       q.CisID,
		DataSources: sources,
	}
}
