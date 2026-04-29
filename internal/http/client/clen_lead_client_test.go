package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCLENLeadClient_GetLeads_HappyPath(t *testing.T) {
	var gotHeaders http.Header
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		gotQuery = r.URL.RawQuery
		if !strings.HasSuffix(r.URL.Path, "/v1/leads/customer-level") {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		payload := []map[string]any{
			{
				"ip_id":  123456,
				"id_num": "1309391174651",
				"list_lead": []map[string]any{
					{
						"lead_id":           "lead_12345",
						"lead_tp":           "Lead For Sales",
						"lead_st":           "NEW",
						"cntnt_id":          "Campaign 2025",
						"csvm_cmpgn_cd":     "active",
						"cmpgn_nm":          "Campaign 2025",
						"exp_dt":            "2025-06-30",
						"asgn_dt":           "2025-06-30",
						"show_sla_cnt_down": true,
						"fnl_scor":          "85.5",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	c := NewCLENLeadClient(CLENLeadConfig{
		BaseURL:       srv.URL,
		APIKey:        "test-key",
		AppIdentifier: "845",
		Timeout:       2 * time.Second,
		RetryCount:    1,
		ExpireFilter:  "true",
	})

	leads, err := c.GetLeads(context.Background(), "123456", LeadQueryParams{
		Channel:    "salesforce",
		Placements: []string{"wsaHomeBanner", "wsaSplash"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(leads) != 1 || leads[0].LeadID != "lead_12345" {
		t.Fatalf("unexpected leads: %+v", leads)
	}
	if leads[0].FinalScore != "85.5" {
		t.Errorf("FinalScore not mapped from fnl_scor: got %q want %q", leads[0].FinalScore, "85.5")
	}
	if gotHeaders.Get("X-Api-Key") != "test-key" {
		t.Errorf("missing X-Api-Key header")
	}
	if !strings.Contains(gotQuery, "ip_id=123456") {
		t.Errorf("ip_id query param not set: %q", gotQuery)
	}
	if !strings.Contains(gotQuery, "exp_f=true") {
		t.Errorf("exp_f=true missing, got %q", gotQuery)
	}
	if !strings.Contains(gotQuery, "chnl=salesforce") {
		t.Errorf("chnl not forwarded, got %q", gotQuery)
	}
	if !strings.Contains(gotQuery, "placement=wsaHomeBanner") || !strings.Contains(gotQuery, "placement=wsaSplash") {
		t.Errorf("placement[] not forwarded as repeated params, got %q", gotQuery)
	}
	if strings.Contains(gotQuery, "channel=") {
		t.Errorf("must not send `channel=`, upstream expects `chnl=`, got %q", gotQuery)
	}
}

func TestCLENLeadClient_GetLeads_OptionalParamsOmitted(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewCLENLeadClient(CLENLeadConfig{BaseURL: srv.URL, APIKey: "k", ExpireFilter: "true"})
	_, _ = c.GetLeads(context.Background(), "1", LeadQueryParams{Placements: nil})

	if strings.Contains(gotQuery, "chnl=") {
		t.Errorf("chnl must be omitted when empty, got %q", gotQuery)
	}
	if strings.Contains(gotQuery, "placement=") {
		t.Errorf("placement must be omitted when empty, got %q", gotQuery)
	}
}

func TestCLENLeadClient_GetLeads_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewCLENLeadClient(CLENLeadConfig{BaseURL: srv.URL, APIKey: "k"})
	leads, err := c.GetLeads(context.Background(), "1", LeadQueryParams{Placements: nil})
	if err != nil || leads != nil {
		t.Fatalf("want nil leads, got %v err=%v", leads, err)
	}
}

func TestCLENLeadClient_GetLeads_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"40100"}}`))
	}))
	defer srv.Close()

	c := NewCLENLeadClient(CLENLeadConfig{BaseURL: srv.URL, APIKey: "k", RetryCount: 1})
	_, err := c.GetLeads(context.Background(), "1", LeadQueryParams{Placements: nil})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want 401 error, got %v", err)
	}
}

func TestCLENLeadClient_Disabled(t *testing.T) {
	c := NewCLENLeadClient(CLENLeadConfig{}) // empty config
	if c != nil {
		t.Fatal("expected nil client for empty config")
	}
	leads, err := (*CLENLeadClient)(nil).GetLeads(context.Background(), "1", LeadQueryParams{Placements: nil})
	if err != nil || leads != nil {
		t.Fatalf("nil client should no-op: %v %v", leads, err)
	}
}

func TestCLENLeadClient_NonNumericCIS(t *testing.T) {
	c := NewCLENLeadClient(CLENLeadConfig{BaseURL: "http://x", APIKey: "k"})
	_, err := c.GetLeads(context.Background(), "abc", LeadQueryParams{Placements: nil})
	if err == nil {
		t.Fatal("expected error on non-numeric cisID")
	}
}

func TestCLENLeadClient_ExpireFilterOmitted(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewCLENLeadClient(CLENLeadConfig{BaseURL: srv.URL, APIKey: "k" /* ExpireFilter left empty */})
	_, _ = c.GetLeads(context.Background(), "1", LeadQueryParams{Placements: nil})

	if strings.Contains(gotQuery, "exp_f") {
		t.Errorf("exp_f should be omitted when ExpireFilter is empty, got %q", gotQuery)
	}
}
