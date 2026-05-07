package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainrepo "kbank-ecms/internal/domain/repository"
	httpclient "kbank-ecms/internal/http/client"
)

// TestCLENCustomerProfileRepository_NilClient_FeatureOff verifies the
// feature-off path: when constructed with a nil underlying client, both
// methods must succeed and return empty/nil values so the delivery service
// can proceed without CLEN profile data.
func TestCLENCustomerProfileRepository_NilClient_FeatureOff(t *testing.T) {
	t.Parallel()

	r := NewCLENCustomerProfileRepository(nil)
	require.NotNil(t, r)

	profile, err := r.GetCustomerProfile(context.Background(), domainrepo.CustomerProfileQuery{CisID: "123"})
	require.NoError(t, err)
	assert.Nil(t, profile)

	raw, err := r.GetCustomerProfileRaw(context.Background(), domainrepo.CustomerProfileQuery{CisID: "123"})
	require.NoError(t, err)
	require.NotNil(t, raw)
	assert.Equal(t, 200, raw.StatusCode)
	assert.JSONEq(t, `{}`, string(raw.Body))
}

// TestCLENCustomerProfileRepository_NilReceiver covers the defensive
// nil-receiver guard on both methods (no-op semantics).
func TestCLENCustomerProfileRepository_NilReceiver(t *testing.T) {
	t.Parallel()

	var r *CLENCustomerProfileRepository

	profile, err := r.GetCustomerProfile(context.Background(), domainrepo.CustomerProfileQuery{CisID: "1"})
	require.NoError(t, err)
	assert.Nil(t, profile)

	raw, err := r.GetCustomerProfileRaw(context.Background(), domainrepo.CustomerProfileQuery{CisID: "1"})
	require.NoError(t, err)
	require.NotNil(t, raw)
	assert.Equal(t, 200, raw.StatusCode)
	assert.JSONEq(t, `{}`, string(raw.Body))
}

// TestCLENCustomerProfileRepository_GetCustomerProfile_DelegatesAndMapsQuery
// verifies the adapter forwards CisID + DataSources (verbatim mapping) to
// the HTTP client and decodes the upstream response into the domain entity.
func TestCLENCustomerProfileRepository_GetCustomerProfile_DelegatesAndMapsQuery(t *testing.T) {
	t.Parallel()

	var gotBody httpclient.CustomerProfileQueryRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"cis_id":"123","results":[{"datasource":"cst_info","status":"success","data":{"foo":"bar"}}],"total_sources_queried":1,"total_sources_success":1}`))
	}))
	defer srv.Close()

	c := httpclient.NewCLENCustomerProfileClient(httpclient.CLENCustomerProfileConfig{BaseURL: srv.URL, APIKey: "k"})
	r := NewCLENCustomerProfileRepository(c)

	q := domainrepo.CustomerProfileQuery{
		CisID: "123",
		DataSources: []domainrepo.CustomerProfileDataSource{
			{Datasource: "cst_info", RequiredFields: []string{"foo", "bar"}},
		},
	}
	profile, err := r.GetCustomerProfile(context.Background(), q)
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "123", profile.CisID)
	assert.Equal(t, "bar", profile.Sources["cst_info"]["foo"])

	// Domain query must round-trip into the wire-level request unchanged.
	assert.Equal(t, "123", gotBody.CisID)
	require.Len(t, gotBody.DataSources, 1)
	assert.Equal(t, "cst_info", gotBody.DataSources[0].Datasource)
	assert.ElementsMatch(t, []string{"foo", "bar"}, gotBody.DataSources[0].RequiredFields)
}

// TestCLENCustomerProfileRepository_GetCustomerProfileRaw_PreservesUpstream
// verifies the raw pass-through preserves status code and body bytes
// verbatim (no decoding, no field renaming).
func TestCLENCustomerProfileRepository_GetCustomerProfileRaw_PreservesUpstream(t *testing.T) {
	t.Parallel()

	upstreamBody := `{"results":[{"datasource":"x","status":"success","data":{"y":1}}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(upstreamBody))
	}))
	defer srv.Close()

	c := httpclient.NewCLENCustomerProfileClient(httpclient.CLENCustomerProfileConfig{BaseURL: srv.URL, APIKey: "k"})
	r := NewCLENCustomerProfileRepository(c)

	raw, err := r.GetCustomerProfileRaw(context.Background(), domainrepo.CustomerProfileQuery{
		CisID:       "123",
		DataSources: []domainrepo.CustomerProfileDataSource{{Datasource: "x", RequiredFields: []string{"y"}}},
	})
	require.NoError(t, err)
	require.NotNil(t, raw)
	assert.Equal(t, 200, raw.StatusCode)
	assert.JSONEq(t, upstreamBody, string(raw.Body))
}

// TestCLENCustomerProfileRepository_RawNon2xxPropagated verifies the raw
// pass-through forwards non-2xx upstream responses without converting them
// to Go errors — the calling controller is expected to forward the same
// shape to its own caller.
func TestCLENCustomerProfileRepository_RawNon2xxPropagated(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c := httpclient.NewCLENCustomerProfileClient(httpclient.CLENCustomerProfileConfig{BaseURL: srv.URL, APIKey: "k"})
	r := NewCLENCustomerProfileRepository(c)

	raw, err := r.GetCustomerProfileRaw(context.Background(), domainrepo.CustomerProfileQuery{CisID: "123"})
	require.NoError(t, err, "raw passthrough must not Go-error on non-2xx")
	require.NotNil(t, raw)
	assert.Equal(t, http.StatusNotFound, raw.StatusCode)
	assert.JSONEq(t, `{"error":"not found"}`, string(raw.Body))
}

// TestCLENCustomerProfileRepository_RawTransportErrorPropagated verifies the
// adapter surfaces transport-level errors from the underlying HTTP client.
func TestCLENCustomerProfileRepository_RawTransportErrorPropagated(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // immediately closed → connection refused on first call

	c := httpclient.NewCLENCustomerProfileClient(httpclient.CLENCustomerProfileConfig{BaseURL: srv.URL, APIKey: "k"})
	r := NewCLENCustomerProfileRepository(c)

	_, err := r.GetCustomerProfileRaw(context.Background(), domainrepo.CustomerProfileQuery{CisID: "123"})
	assert.Error(t, err, "transport failure must propagate")
}
