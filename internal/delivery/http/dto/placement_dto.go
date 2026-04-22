package dto

import (
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
)

// CreatePlacementRequest is the request body for POST /placements.
type CreatePlacementRequest struct {
	PlacementName string    `json:"placementName" binding:"required,max=255"`
	ChannelID     uuid.UUID `json:"channelId"     binding:"required"`
}

// UpdatePlacementRequest is the request body for PUT /placements/:id.
type UpdatePlacementRequest struct {
	PlacementName string    `json:"placementName" binding:"required,max=255"`
	ChannelID     uuid.UUID `json:"channelId"     binding:"required"`
}

// PlacementResponse is the response body for placement endpoints.
// Channel is always embedded so callers never need a second round-trip.
type PlacementResponse struct {
	ID            uuid.UUID       `json:"id"`
	PlacementName string          `json:"placementName"`
	Channel       ChannelResponse `json:"channel"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

// ToPlacementResponse converts a Placement entity to a PlacementResponse DTO.
// placement.Channel must be preloaded before calling this function.
func ToPlacementResponse(p *entity.Placement) PlacementResponse {
	var ch ChannelResponse
	if p.Channel != nil {
		ch = ToChannelResponse(p.Channel)
	}
	return PlacementResponse{
		ID:            p.ID,
		PlacementName: p.PlacementName,
		Channel:       ch,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}
