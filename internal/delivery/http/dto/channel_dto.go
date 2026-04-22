package dto

import (
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
)

// CreateChannelRequest is the request body for POST /channels.
type CreateChannelRequest struct {
	ChannelName string `json:"channelName" binding:"required,max=255"`
}

// UpdateChannelRequest is the request body for PUT /channels/:id.
type UpdateChannelRequest struct {
	ChannelName string `json:"channelName" binding:"required,max=255"`
}

// ChannelResponse is the response body for channel endpoints.
type ChannelResponse struct {
	ID          uuid.UUID `json:"id"`
	ChannelName string    `json:"channelName"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ToChannelResponse converts a Channel entity to a ChannelResponse DTO.
func ToChannelResponse(c *entity.Channel) ChannelResponse {
	return ChannelResponse{
		ID:          c.ID,
		ChannelName: c.ChannelName,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
