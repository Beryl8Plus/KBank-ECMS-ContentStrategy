package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCLENCustomerProfileClient_GetCustomerProfile_HappyPath(t *testing.T) {
	var gotHeaders http.Header
	var gotMethod string
	var gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"cis_id": "123456",
			"results": []map[string]any{
				{
					"datasource":    "cst_info_prfl_dly",
					"status":        "success",
					"data":          map[string]any{"wlth_seg_cd": "W3"},
					"error_message": nil,
				},
				{
					"datasource":    "centralized_sandbox",
					"status":        "success",
					"data":          map[string]any{"WEALTH_SEGMENT": "GOLD"},
					"error_message": nil,
				},
			},
			"total_sources_queried": 2,
			"total_sources_success": 2,
		})
	}))
	defer srv.Close()

	c := NewCLENCustomerProfileClient(CLENCustomerProfileConfig{
		BaseURL:       srv.URL,
		APIKey:        "test-key",
		AppIdentifier: "808",
		Timeout:       2 * time.Second,
		RetryCount:    1,
	})

	body := CustomerProfileQueryRequest{
		CisID: "123456",
		DataSources: []CustomerProfileDataSourceReq{
			{Datasource: "cst_info_prfl_dly", RequiredFields: []string{"wlth_seg_cd"}},
			{Datasource: "centralized_sandbox", RequiredFields: []string{"WEALTH_SEGMENT"}},
		},
	}
	profile, err := c.GetCustomerProfile(context.Background(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile == nil || profile.CisID != "123456" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
	if got := profile.Sources["cst_info_prfl_dly"]["wlth_seg_cd"]; got != "W3" {
		t.Errorf("missing wlth_seg_cd, got %v", got)
	}
	if got := profile.Sources["centralized_sandbox"]["WEALTH_SEGMENT"]; got != "GOLD" {
		t.Errorf("missing WEALTH_SEGMENT, got %v", got)
	}

	// Headers
	if gotHeaders.Get("X-Api-Key") != "test-key" {
		t.Errorf("missing X-Api-Key header")
	}
	if gotHeaders.Get("Request-Application-Identifier") != "808" {
		t.Errorf("missing Request-Application-Identifier")
	}
	if gotHeaders.Get("Request-Identifier") == "" {
		t.Errorf("missing Request-Identifier (uuid)")
	}
	if gotHeaders.Get("Request-Datetime") == "" {
		t.Errorf("missing Request-Datetime")
	}
	if !strings.Contains(gotHeaders.Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type must be application/json, got %q", gotHeaders.Get("Content-Type"))
	}

	// Method + path
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/v1/customers/query") {
		t.Errorf("unexpected path %q", gotPath)
	}

	// Body
	var sentBody CustomerProfileQueryRequest
	if err := json.Unmarshal(gotBody, &sentBody); err != nil {
		t.Fatalf("decode sent body: %v", err)
	}
	if sentBody.CisID != "123456" {
		t.Errorf("body cis_id mismatch, got %q", sentBody.CisID)
	}
	if len(sentBody.DataSources) != 2 {
		t.Errorf("expected 2 data_sources, got %d", len(sentBody.DataSources))
	}
}

func TestCLENCustomerProfileClient_GetCustomerProfileRaw_NonNumericCISAllowed(t *testing.T) {
	// Unlike Lead, the customer-profile endpoint accepts non-numeric cis_id
	// (treated as opaque string). Verify the call still goes through.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"cis_id":"abc","results":[],"total_sources_queried":0,"total_sources_success":0}`))
	}))
	defer srv.Close()

	c := NewCLENCustomerProfileClient(CLENCustomerProfileConfig{BaseURL: srv.URL, APIKey: "k"})
	raw, err := c.GetCustomerProfileRaw(context.Background(), CustomerProfileQueryRequest{CisID: "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw.StatusCode != 200 {
		t.Errorf("want 200, got %d", raw.StatusCode)
	}
}

func TestCLENCustomerProfileClient_GetCustomerProfile_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"40100"}}`))
	}))
	defer srv.Close()

	c := NewCLENCustomerProfileClient(CLENCustomerProfileConfig{BaseURL: srv.URL, APIKey: "k", RetryCount: 1})
	_, err := c.GetCustomerProfile(context.Background(), CustomerProfileQueryRequest{CisID: "1"})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want 401 error, got %v", err)
	}
}

func TestCLENCustomerProfileClient_GetCustomerProfileRaw_PassesThroughNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"40400"}}`))
	}))
	defer srv.Close()

	c := NewCLENCustomerProfileClient(CLENCustomerProfileConfig{BaseURL: srv.URL, APIKey: "k", RetryCount: 1})
	raw, err := c.GetCustomerProfileRaw(context.Background(), CustomerProfileQueryRequest{CisID: "1"})
	if err != nil {
		t.Fatalf("raw must not return Go error on non-2xx, got %v", err)
	}
	if raw.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 propagated, got %d", raw.StatusCode)
	}
}

func TestCLENCustomerProfileClient_OnlySuccessfulSourcesIncluded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"cis_id":"123",
			"results":[
				{"datasource":"a","status":"success","data":{"x":1},"error_message":null},
				{"datasource":"b","status":"not_found","data":null,"error_message":null},
				{"datasource":"c","status":"error","data":null,"error_message":"boom"}
			],
			"total_sources_queried":3,
			"total_sources_success":1
		}`))
	}))
	defer srv.Close()

	c := NewCLENCustomerProfileClient(CLENCustomerProfileConfig{BaseURL: srv.URL, APIKey: "k"})
	profile, err := c.GetCustomerProfile(context.Background(), CustomerProfileQueryRequest{CisID: "123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile == nil {
		t.Fatal("want non-nil profile")
	}
	if len(profile.Sources) != 1 {
		t.Errorf("only successful sources should be kept, got %v", profile.Sources)
	}
	if _, ok := profile.Sources["a"]; !ok {
		t.Errorf("missing source 'a'")
	}
}

func TestCLENCustomerProfileClient_Disabled(t *testing.T) {
	c := NewCLENCustomerProfileClient(CLENCustomerProfileConfig{}) // empty config
	if c != nil {
		t.Fatal("expected nil client for empty config")
	}
	profile, err := (*CLENCustomerProfileClient)(nil).GetCustomerProfile(context.Background(), CustomerProfileQueryRequest{CisID: "1"})
	if err != nil || profile != nil {
		t.Fatalf("nil client should no-op: %v %v", profile, err)
	}
	raw, err := (*CLENCustomerProfileClient)(nil).GetCustomerProfileRaw(context.Background(), CustomerProfileQueryRequest{CisID: "1"})
	if err != nil || raw == nil || raw.StatusCode != 200 {
		t.Fatalf("nil client raw should return 200 default: %v %v", raw, err)
	}
}

func TestCLENCustomerProfileClient_EmptyCisIDIsNoop(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	c := NewCLENCustomerProfileClient(CLENCustomerProfileConfig{BaseURL: srv.URL, APIKey: "k"})
	raw, err := c.GetCustomerProfileRaw(context.Background(), CustomerProfileQueryRequest{CisID: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("upstream must not be called when cis_id is empty")
	}
	if raw == nil || raw.StatusCode != 200 {
		t.Errorf("expected 200 default, got %+v", raw)
	}
}
