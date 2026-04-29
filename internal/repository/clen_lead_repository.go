package repository

import (
	"context"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	httpclient "kbank-ecms/internal/http/client"
)

// CLENLeadRepository adapts *client.CLENLeadClient to domain LeadRepository.
type CLENLeadRepository struct {
	client *httpclient.CLENLeadClient
}

// Compile-time interface check.
var _ domainrepo.LeadRepository = (*CLENLeadRepository)(nil)

// NewCLENLeadRepository accepts a nil client (feature-off). In that case
// GetByCIS returns an empty slice and a nil error, letting the delivery
// service proceed without lead data.
func NewCLENLeadRepository(c *httpclient.CLENLeadClient) *CLENLeadRepository {
	return &CLENLeadRepository{client: c}
}

func (r *CLENLeadRepository) GetLeads(ctx context.Context, q domainrepo.LeadQuery) ([]entity.Lead, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	return r.client.GetLeads(ctx, q.CisID, httpclient.LeadQueryParams{
		Channel:    q.Channel,
		Placements: q.Placements,
	})
}
