package entity

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"kbank-ecms/pkg/ctxconsts"
)

func TestBaseModel_BeforeCreate(t *testing.T) {
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), ctxconsts.UserIDKey, uid)

	model := &BaseModel{}
	db, _ := gorm.Open(nil, &gorm.Config{DryRun: true})
	tx := db.WithContext(ctx)

	err := model.BeforeCreate(tx)
	assert.NoError(t, err)
	assert.NotNil(t, model.CreatedBy)
	assert.Equal(t, uid, *model.CreatedBy)
	assert.NotNil(t, model.UpdatedBy)
	assert.Equal(t, uid, *model.UpdatedBy)
}

func TestBaseModel_BeforeUpdate(t *testing.T) {
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), ctxconsts.UserIDKey, uid.String())

	model := &BaseModel{}
	db, _ := gorm.Open(nil, &gorm.Config{DryRun: true})
	tx := db.WithContext(ctx)

	err := model.BeforeUpdate(tx)
	assert.NoError(t, err)
	assert.NotNil(t, model.UpdatedBy)
	assert.Equal(t, uid, *model.UpdatedBy)
}

func TestBaseModel_getUserID(t *testing.T) {
	model := &BaseModel{}
	uid := uuid.New()

	t.Run("UUID in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ctxconsts.UserIDKey, uid)
		got := model.getUserID(ctx)
		assert.Equal(t, uid, *got)
	})

	t.Run("String UUID in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ctxconsts.UserIDKey, uid.String())
		got := model.getUserID(ctx)
		assert.Equal(t, uid, *got)
	})

	t.Run("Empty context", func(t *testing.T) {
		got := model.getUserID(context.TODO())
		assert.Nil(t, got)
	})

	t.Run("Missing key in context", func(t *testing.T) {
		got := model.getUserID(context.Background())
		assert.Nil(t, got)
	})
}
