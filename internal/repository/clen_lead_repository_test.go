package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainrepo "kbank-ecms/internal/domain/repository"
	httpclient "kbank-ecms/internal/http/client"
)

// TestCLENLeadRepository_NilClient_FeatureOff verifies the feature-off path:
// when constructed with a nil underlying client, GetLeads returns nil/nil so
// the delivery service can proceed without lead data.
func TestCLENLeadRepository_NilClient_FeatureOff(t *testing.T) {
	t.Parallel()

	r := NewCLENLeadRepository(nil)
	require.NotNil(t, r)

	leads, err := r.GetLeads(context.Background(), domainrepo.LeadQuery{
		CisID:      "1",
		Channel:    "salesforce",
		Placements: []string{"wsaHome"},
	})
	require.NoError(t, err)
	assert.Nil(t, leads)
}

// TestCLENLeadRepository_NilReceiver covers the defensive nil-receiver guard.
func TestCLENLeadRepository_NilReceiver(t *testing.T) {
	t.Parallel()

	var r *CLENLeadRepository
	leads, err := r.GetLeads(context.Background(), domainrepo.LeadQuery{CisID: "1"})
	require.NoError(t, err)
	assert.Nil(t, leads)
}

// TestCLENLeadRepository_GetLeads_DelegatesQueryParams verifies the adapter
// forwards LeadQuery (CisID + Channel + Placements) to the HTTP client and
// returns the decoded leads.
func TestCLENLeadRepository_GetLeads_DelegatesQueryParams(t *testing.T) {
	t.Parallel()

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotQuery = req.URL.RawQuery
		_, _ = w.Write([]byte(`[{"ip_id":1,"id_num":"x","list_lead":[{"lead_id":"L1","lead_tp":"Sales","lead_st":"NEW"}]}]`))
	}))
	defer srv.Close()

	c := httpclient.NewCLENLeadClient(httpclient.CLENLeadConfig{BaseURL: srv.URL, APIKey: "k"})
	r := NewCLENLeadRepository(c)

	leads, err := r.GetLeads(context.Background(), domainrepo.LeadQuery{
		CisID:      "1",
		Channel:    "salesforce",
		Placements: []string{"wsaHomeBanner", "wsaSplash"},
	})
	require.NoError(t, err)
	require.Len(t, leads, 1)
	assert.Equal(t, "L1", leads[0].LeadID)

	// Verify domain-level query maps to the upstream query string.
	assert.Contains(t, gotQuery, "ip_id=1")
	assert.Contains(t, gotQuery, "chnl=salesforce")
	assert.Contains(t, gotQuery, "placement=wsaHomeBanner")
	assert.Contains(t, gotQuery, "placement=wsaSplash")
}

// TestCLENLeadRepository_GetLeads_TransportErrorPropagated verifies the
// adapter surfaces upstream transport errors (e.g. non-numeric cisID).
func TestCLENLeadRepository_GetLeads_TransportErrorPropagated(t *testing.T) {
	t.Parallel()

	c := httpclient.NewCLENLeadClient(httpclient.CLENLeadConfig{BaseURL: "http://example", APIKey: "k"})
	r := NewCLENLeadRepository(c)

	_, err := r.GetLeads(context.Background(), domainrepo.LeadQuery{CisID: "not-a-number"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not numeric") || strings.Contains(err.Error(), "not-a-number"),
		"expected non-numeric cisID error, got %v", err)
}

// TestCLENLeadRepository_GetLeads_EmptyResults handles the no-leads path:
// upstream returns []; adapter returns nil leads + nil error.
func TestCLENLeadRepository_GetLeads_EmptyResults(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := httpclient.NewCLENLeadClient(httpclient.CLENLeadConfig{BaseURL: srv.URL, APIKey: "k"})
	r := NewCLENLeadRepository(c)

	leads, err := r.GetLeads(context.Background(), domainrepo.LeadQuery{CisID: "1"})
	require.NoError(t, err)
	assert.Nil(t, leads)
}
