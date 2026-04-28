package service

import (
	"context"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

// StubExternalAttributeAPIClient is a placeholder implementation of
// ExternalAttributeAPIClient. Replace this with the real HTTP client once the
// CLEN endpoint details are available.
type StubExternalAttributeAPIClient struct{}

// NewStubExternalAttributeAPIClient creates a StubExternalAttributeAPIClient.
func NewStubExternalAttributeAPIClient() *StubExternalAttributeAPIClient {
	return &StubExternalAttributeAPIClient{}
}

// FetchAllAttributes returns an empty list and logs a warning. The sync job
// will be a no-op until a real client is wired in.
func (c *StubExternalAttributeAPIClient) FetchAllAttributes(ctx context.Context) ([]*ExternalAttributeSchema, error) {
	logger.LSystem(ctx, entity.SystemLog{
		Service: "ATTRIBUTE-SYNC",
		Level:   "WARN",
		Message: "FetchAllAttributes called on stub client — no data fetched",
	})
	return nil, nil
}
