package entity

// Channel represents a delivery channel for content placements.
//
// Table: channels
type Channel struct {
	BaseModel
	ChannelName string `gorm:"size:255" json:"channelName"`
}
