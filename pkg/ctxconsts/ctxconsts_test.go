package ctxconsts

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestGetUserID(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, id)
	got, ok := GetUserID(ctx)
	if !ok || got == nil || *got != id {
		t.Errorf("GetUserID round-trip failed: got=%v ok=%v", got, ok)
	}

	// missing key → not ok
	got2, ok2 := GetUserID(context.Background())
	if ok2 || got2 != nil {
		t.Errorf("missing key: got=%v ok=%v", got2, ok2)
	}

	// wrong type → not ok
	bad := context.WithValue(context.Background(), UserIDKey, "not-a-uuid")
	if _, ok := GetUserID(bad); ok {
		t.Error("wrong type should return ok=false")
	}
}

func TestGetCisID(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), CisIDKey, "C0001")
	if v, ok := GetCisID(ctx); !ok || v != "C0001" {
		t.Errorf("GetCisID: %q ok=%v", v, ok)
	}
	if _, ok := GetCisID(context.Background()); ok {
		t.Error("missing CisID should return ok=false")
	}
}

func TestGetClientID(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ClientIDKey, "client-1")
	if v, ok := GetClientID(ctx); !ok || v != "client-1" {
		t.Errorf("GetClientID: %q ok=%v", v, ok)
	}
	if _, ok := GetClientID(context.Background()); ok {
		t.Error("missing ClientID should return ok=false")
	}
}

func TestGetScopes(t *testing.T) {
	t.Parallel()
	scopes := []string{"read", "write"}
	ctx := context.WithValue(context.Background(), ScopesKey, scopes)
	if v, ok := GetScopes(ctx); !ok || len(v) != 2 || v[0] != "read" {
		t.Errorf("GetScopes: %v ok=%v", v, ok)
	}
	if _, ok := GetScopes(context.Background()); ok {
		t.Error("missing scopes should return ok=false")
	}
}

func TestGetDB(t *testing.T) {
	t.Parallel()
	db := &gorm.DB{}
	ctx := context.WithValue(context.Background(), DBKey, db)
	if v, ok := GetDB(ctx); !ok || v != db {
		t.Errorf("GetDB: %v ok=%v", v, ok)
	}
	if _, ok := GetDB(context.Background()); ok {
		t.Error("missing DB should return ok=false")
	}
}

func TestCorrelationID_RoundTrip(t *testing.T) {
	t.Parallel()
	ctx := SetCorrelationID(context.Background(), "abc-123")
	if got := GetCorrelationID(ctx); got != "abc-123" {
		t.Errorf("CorrelationID round-trip failed: %q", got)
	}
	if got := GetCorrelationID(context.Background()); got != "" {
		t.Errorf("missing CorrelationID should return empty string, got %q", got)
	}
}
