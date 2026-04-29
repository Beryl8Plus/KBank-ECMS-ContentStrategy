package entity

import "github.com/google/uuid"

// OAuth2Client represents an OAuth2 client allowed to obtain tokens via
// the Client Credentials Flow. Each client is linked to a Profile that
// determines its scopes/permissions.
//
// Table: oauth2_clients
type OAuth2Client struct {
	BaseModel
	ClientID     string    `gorm:"size:255;uniqueIndex;not null" json:"clientId"`
	ClientSecret string    `gorm:"size:255;not null"             json:"-"`
	ProfileID    uuid.UUID `gorm:"type:uuid;not null"            json:"profileId"`
	Description  string    `gorm:"size:500"                      json:"description"`
	IsActive     bool      `gorm:"default:true"                  json:"isActive"`

	Profile *Profile `gorm:"foreignKey:ProfileID;references:ID" json:"profile,omitempty"`
}

// TableName overrides the default GORM naming convention which would otherwise
// produce "o_auth2_clients" due to the leading "OAuth" acronym.
func (OAuth2Client) TableName() string {
	return "oauth2_clients"
}
