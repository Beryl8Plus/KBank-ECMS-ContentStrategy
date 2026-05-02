package entity

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kbank-ecms/pkg/ctxconsts"
)

// BaseModel provides common audit fields for all database entities.
// Embed this struct in any GORM model that requires:
//   - UUID primary key (auto-generated)
//   - Created/Updated timestamps (auto-managed by GORM)
//   - Created/Updated by user references (nullable FK → users.id)
//   - Soft-delete support
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `gorm:"type:timestamptz;autoCreateTime;<-:create"      json:"createdAt"`
	CreatedBy *uuid.UUID     `gorm:"type:uuid;<-:create"                            json:"createdBy"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;autoUpdateTime;<-"             json:"updatedAt"`
	UpdatedBy *uuid.UUID     `gorm:"type:uuid;<-"                                   json:"updatedBy"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                          json:"-"`

	// These fields are not stored in the database but can be populated in code for convenience.
	CreatedByUser *User `gorm:"-"                                              swaggerignore:"true" json:"createdByUser,omitempty"`
	UpdatedByUser *User `gorm:"-"                                              swaggerignore:"true" json:"updatedByUser,omitempty"`
}

// BeforeCreate is a GORM hook to set CreatedBy and UpdatedBy automatically.
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if uid := b.getUserID(tx.Statement.Context); uid != nil {
		b.CreatedBy = uid
		b.UpdatedBy = uid
		tx.Statement.SetColumn("CreatedBy", uid)
		tx.Statement.SetColumn("UpdatedBy", uid)
	}
	return nil
}

// BeforeUpdate is a GORM hook to set UpdatedBy automatically.
func (b *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	if uid := b.getUserID(tx.Statement.Context); uid != nil {
		b.UpdatedBy = uid
		tx.Statement.SetColumn("UpdatedBy", uid)
	}
	return nil
}

// BeforeDelete is a GORM hook to stamp UpdatedBy on soft-delete.
func (b *BaseModel) BeforeDelete(tx *gorm.DB) error {
	if uid := b.getUserID(tx.Statement.Context); uid != nil {
		b.UpdatedBy = uid
		tx.Statement.SetColumn("UpdatedBy", uid)
	}
	return nil
}

// getUserID extracts the current user ID from the context.
func (b *BaseModel) getUserID(ctx context.Context) *uuid.UUID {
	if ctx == nil {
		return nil
	}
	// Try direct UUID (set in context as uuid.UUID)
	if v, ok := ctx.Value(ctxconsts.UserIDKey).(uuid.UUID); ok {
		return &v
	}
	// Try parsing from string (set in context as string from JWT or header)
	if v, ok := ctx.Value(ctxconsts.UserIDKey).(string); ok {
		if id, err := uuid.Parse(v); err == nil {
			return &id
		}
	}
	return nil
}
