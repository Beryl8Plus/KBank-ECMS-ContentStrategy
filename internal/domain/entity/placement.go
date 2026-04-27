package entity

import "github.com/google/uuid"

// Placement represents a named content placement slot within a channel.
//
// Table: placements
type Placement struct {
	BaseModel
	PlacementName string    `gorm:"size:255;not null;uniqueIndex:idx_channel_placement_name"  json:"placementName"`
	ChannelID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_channel_placement_name" json:"channelId"`
	Channel       *Channel  `gorm:"foreignKey:ChannelID;references:ID"                        json:"channel,omitempty"`
}
