package entity

import "github.com/google/uuid"

// Placement represents a named content placement slot within a channel.
//
// Table: placements
type Placement struct {
	BaseModel
	PlacementName string    `gorm:"size:255"                           json:"placementName"`
	ChannelID     uuid.UUID `gorm:"type:uuid;not null"                 json:"channelId"`
	Channel       *Channel  `gorm:"foreignKey:ChannelID;references:ID" json:"channel,omitempty"`
}
