package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/pkg/util/httpclient"
)

// CLENCustomerProfileConfig carries the knobs for talking to the CLEN
// Customer Dynamic Query API.
type CLENCustomerProfileConfig struct {
	BaseURL       string        // e.g. https://apimdataappseadev002.../clen/customer-inquiry
	Path          string        // request path appended to BaseURL (e.g. /v1/customers/query)
	APIKey        string        // X-Api-Key header value
	AppIdentifier string        // Request-Application-Identifier header value (KBTG-assigned)
	Timeout       time.Duration // per-request timeout
	RetryCount    int
}

// Enabled reports whether the client is configured enough to make real calls.
func (c CLENCustomerProfileConfig) Enabled() bool {
	return c.BaseURL != "" && c.APIKey != ""
}

// CLENCustomerProfileClient calls `POST /v1/customers/query` and exposes both
// a verbatim pass-through (GetCustomerProfileRaw) and a decoded view
// (GetCustomerProfile) used by internal enrichment.
type CLENCustomerProfileClient struct {
	cfg        CLENCustomerProfileConfig
	restClient *httpclient.RestClient
}

// defaultCLENCustomerProfilePath is used when CLENCustomerProfileConfig.Path is empty.
const defaultCLENCustomerProfilePath = "/v1/customers/query"

// NewCLENCustomerProfileClient returns nil when the config is not enabled —
// callers must treat a nil *CLENCustomerProfileClient as "feature disabled".
// An empty Path is filled with the default upstream path so callers don't
// have to.
func NewCLENCustomerProfileClient(cfg CLENCustomerProfileConfig) *CLENCustomerProfileClient {
	if !cfg.Enabled() {
		return nil
	}
	if cfg.Path == "" {
		cfg.Path = defaultCLENCustomerProfilePath
	}
	return &CLENCustomerProfileClient{
		cfg: cfg,
		restClient: httpclient.NewRestClient(httpclient.Config{
			BaseURL:    cfg.BaseURL,
			Timeout:    cfg.Timeout,
			RetryCount: cfg.RetryCount,
		}),
	}
}

// CustomerProfileQueryRequest is the JSON body sent to CLEN.
type CustomerProfileQueryRequest struct {
	CisID       string                         `json:"cis_id"`
	DataSources []CustomerProfileDataSourceReq `json:"data_sources"`
}

// CustomerProfileDataSourceReq is one element of CustomerProfileQueryRequest.data_sources.
type CustomerProfileDataSourceReq struct {
	Datasource     string   `json:"datasource"`
	RequiredFields []string `json:"required_fields"`
}

// clenCustomerProfileResponse mirrors the CLEN response wrapper.
type clenCustomerProfileResponse struct {
	CisID               string                      `json:"cis_id"`
	Results             []clenCustomerProfileResult `json:"results"`
	TotalSourcesQueried int                         `json:"total_sources_queried"`
	TotalSourcesSuccess int                         `json:"total_sources_success"`
}

// clenCustomerProfileResult is one entry under "results".
type clenCustomerProfileResult struct {
	Datasource   string         `json:"datasource"`
	Status       string         `json:"status"` // "success" | "not_found" | "error"
	Data         map[string]any `json:"data"`
	ErrorMessage *string        `json:"error_message"`
}

// GetCustomerProfileRaw POSTs the request to CLEN and returns the upstream
// body verbatim (no decoding, no field renaming). Non-2xx responses are NOT
// converted to Go errors — they are returned with their body so the caller
// can forward them as-is. The error return is reserved for transport failures.
func (c *CLENCustomerProfileClient) GetCustomerProfileRaw(ctx context.Context, body CustomerProfileQueryRequest) (*RawResponse, error) {
	if c == nil {
		return &RawResponse{StatusCode: 200, Body: []byte("{}")}, nil
	}
	if body.CisID == "" {
		return &RawResponse{StatusCode: 200, Body: []byte("{}")}, nil
	}

	req := c.restClient.Client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-Api-Key", c.cfg.APIKey).
		SetHeader("Request-Identifier", uuid.NewString()).
		SetHeader("Request-Application-Identifier", c.cfg.AppIdentifier).
		SetHeader("Request-Datetime", time.Now().UTC().Format("2006-01-02T15:04:05.000")).
		SetBody(body)

	resp, err := req.Post(c.cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("call CLEN customer profile: %w", err)
	}
	return &RawResponse{StatusCode: resp.StatusCode(), Body: resp.Bytes()}, nil
}

// GetCustomerProfile is the decoded view of GetCustomerProfileRaw used by
// internal callers (e.g. resolveUserAttrs). Returns nil + nil when the client
// is disabled or CLEN reports no successful sources. Non-2xx responses return
// an error.
func (c *CLENCustomerProfileClient) GetCustomerProfile(ctx context.Context, body CustomerProfileQueryRequest) (*entity.CustomerProfile, error) {
	if c == nil {
		return nil, nil
	}
	if body.CisID == "" {
		return nil, nil
	}

	raw, err := c.GetCustomerProfileRaw(ctx, body)
	if err != nil {
		return nil, err
	}
	if raw.StatusCode >= 400 {
		snippet := string(raw.Body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("CLEN customer profile %d: %s", raw.StatusCode, snippet)
	}
	if len(raw.Body) == 0 {
		return nil, nil
	}

	var decoded clenCustomerProfileResponse
	if err := json.Unmarshal(raw.Body, &decoded); err != nil {
		return nil, fmt.Errorf("decode CLEN customer profile response: %w", err)
	}

	sources := make(map[string]map[string]any, len(decoded.Results))
	for _, r := range decoded.Results {
		if r.Status != "success" || len(r.Data) == 0 {
			continue
		}
		sources[r.Datasource] = r.Data
	}
	if len(sources) == 0 {
		return nil, nil
	}
	return &entity.CustomerProfile{
		CisID:   decoded.CisID,
		Sources: sources,
	}, nil
}
