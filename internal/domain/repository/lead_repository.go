package repository

import (
	"context"

	"kbank-ecms/internal/domain/entity"
)

// LeadQuery carries the optional filters forwarded to the upstream CLEN
// Lead API. CisID is mandatory; Channel (sent as "chnl") and Placements
// (sent as repeated "placement") are forwarded verbatim and may be blank/empty.
type LeadQuery struct {
	CisID      string
	Channel    string
	Placements []string
}

// LeadRepository abstracts the upstream Lead Information provider (CLEN).
type LeadRepository interface {
	// GetLeads returns the leads matching the given query.
	// An empty slice with a nil error means "no leads" — not an error.
	GetLeads(ctx context.Context, q LeadQuery) ([]entity.Lead, error)
}
