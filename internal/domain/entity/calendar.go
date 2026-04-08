package entity

import "kbank-ecms/internal/domain/entity/enums"

// Calendar represents a named calendar source (holidays, personal dates, etc.).
//
// Table: calendars
type Calendar struct {
	BaseModel
	Name     string             `gorm:"size:255"     json:"name"`
	Type     enums.CalendarType `gorm:"size:255"     json:"type"`
	IsActive bool               `gorm:"default:true" json:"isActive"`
}
