package entity

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"kbank-ecms/pkg/ctxconsts"
)

func TestAttribute_ValidOptions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		value   string
		want    map[string]struct{}
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"simple list", `["GOLD","SILVER"]`, map[string]struct{}{"GOLD": {}, "SILVER": {}}, false},
		{"structured list", `[{"value":"A"},{"value":"B"}]`, map[string]struct{}{"A": {}, "B": {}}, false},
		{"invalid", `{"not":"array"}`, nil, true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := &Attribute{}
			if c.value != "" {
				a.Value = datatypes.JSON(c.value)
			}
			got, err := a.ValidOptions()
			if c.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if len(got) != len(c.want) {
				t.Errorf("len mismatch: got %d, want %d", len(got), len(c.want))
			}
			for k := range c.want {
				if _, ok := got[k]; !ok {
					t.Errorf("missing key %q", k)
				}
			}
		})
	}
}

func TestAllModels_NonEmpty(t *testing.T) {
	t.Parallel()
	got := AllModels()
	if len(got) == 0 {
		t.Error("AllModels() returned empty slice")
	}
}

func TestTableNames(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"clen_schema_registry": CLENSchemaRegistry{}.TableName(),
		"oauth2_clients":       OAuth2Client{}.TableName(),
		"profile_permissions":  ProfilePermission{}.TableName(),
		"rules":                Rule{}.TableName(),
		"rule_attributes":      RuleAttribute{}.TableName(),
		"rule_conditions":      RuleCondition{}.TableName(),
	}
	for want, got := range cases {
		if got != want {
			t.Errorf("TableName: got %q, want %q", got, want)
		}
	}
}

func TestBaseModel_BeforeDelete(t *testing.T) {
	t.Parallel()
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), ctxconsts.UserIDKey, uid)

	model := &BaseModel{}
	db, _ := gorm.Open(nil, &gorm.Config{DryRun: true})
	tx := db.WithContext(ctx)

	err := model.BeforeDelete(tx)
	assert.NoError(t, err)
	assert.NotNil(t, model.UpdatedBy)
	assert.Equal(t, uid, *model.UpdatedBy)
}

func TestBaseModel_getUserID_InvalidStringNotPanic(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ctxconsts.UserIDKey, "not-a-uuid")
	got := (&BaseModel{}).getUserID(ctx)
	if got != nil {
		t.Errorf("invalid UUID string should return nil, got %v", got)
	}
}
