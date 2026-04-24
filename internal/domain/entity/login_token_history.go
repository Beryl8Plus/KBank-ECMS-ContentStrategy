package entity

import "time"

// LoginTokenHistory stores authentication token history per user.
//
// Table: login_token_histories
type LoginTokenHistory struct {
	BaseModel
	UserName    string    `gorm:"size:255;uniqueIndex" json:"username"`
	AccessToken string    `gorm:"size:255"             json:"accessToken"`
	ExpireDate  time.Time `gorm:"type:timestamptz"     json:"expireDate"`
}
