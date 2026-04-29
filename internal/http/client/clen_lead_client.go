package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/pkg/util/httpclient"
)

// CLENLeadConfig carries the knobs for talking to CLEN Lead API.
type CLENLeadConfig struct {
	BaseURL       string        // e.g. https://apimdataappseadev002.../clen/core-lead-info-inquiry
	Path          string        // request path appended to BaseURL (e.g. /v1/leads/customer-level)
	APIKey        string        // X-Api-Key header value
	AppIdentifier string        // Request-Application-Identifier header value (KBTG-assigned)
	Timeout       time.Duration // per-request timeout, defaults applied by httpclient
	RetryCount    int

	// ExpireFilter is the literal value sent as the `exp_f` query param.
	// "true"  → only non-expired leads (default)
	// "false" → only expired leads
	// ""      → omit the param (CLEN returns both)
	ExpireFilter string
}

// Enabled reports whether the client is configured enough to make real calls.
func (c CLENLeadConfig) Enabled() bool {
	return c.BaseURL != "" && c.APIKey != ""
}

// CLENLeadClient calls `GET /v1/leads/customer-level` and maps the response
// to domain entities.
type CLENLeadClient struct {
	cfg        CLENLeadConfig
	restClient *httpclient.RestClient
}

// defaultCLENLeadPath is used when CLENLeadConfig.Path is empty.
const defaultCLENLeadPath = "/v1/leads/customer-level"

// NewCLENLeadClient returns nil when the config is not enabled — callers
// must treat a nil *CLENLeadClient as "feature disabled". An empty Path
// is filled with the default upstream path so callers don't have to.
func NewCLENLeadClient(cfg CLENLeadConfig) *CLENLeadClient {
	if !cfg.Enabled() {
		return nil
	}
	if cfg.Path == "" {
		cfg.Path = defaultCLENLeadPath
	}
	return &CLENLeadClient{
		cfg: cfg,
		restClient: httpclient.NewRestClient(httpclient.Config{
			BaseURL:    cfg.BaseURL,
			Timeout:    cfg.Timeout,
			RetryCount: cfg.RetryCount,
		}),
	}
}

// clenLeadGroup mirrors one element of the CLEN customer-level response.
type clenLeadGroup struct {
	IPID  int64      `json:"ip_id"`
	IDNum string     `json:"id_num"`
	Leads []clenLead `json:"list_lead"`
}

type clenLead struct {
	LeadID           string   `json:"lead_id"`
	LeadType         string   `json:"lead_tp"`
	LeadStatus       string   `json:"lead_st"`
	ContentID        string   `json:"cntnt_id"`
	CSVMCampaignCode string   `json:"csvm_cmpgn_cd"`
	RMSTCampaignCode string   `json:"rmst_cmpgn_cd"`
	CampaignName     string   `json:"cmpgn_nm"`
	ExpireDate       string   `json:"exp_dt"`
	AssignedDate     string   `json:"asgn_dt"`
	SLAStartDate     string   `json:"sla_strt_dt"`
	SLAEndDate       string   `json:"sla_end_dt"`
	ShowSLACountDown bool     `json:"show_sla_cnt_down"`
	FinalScore       string   `json:"fnl_scor"`
	Placement        []string `json:"placement"`
}

// nonEmpty filters out blank strings so we don't accidentally send
// ?placement= (empty-value) to the upstream.
func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// LeadQueryParams bundles optional filters forwarded verbatim to CLEN.
// Empty fields are not sent. Placements is sent as a multi-value query param
// (form/explode): ?placement=a&placement=b.
type LeadQueryParams struct {
	Channel    string   // sent as "chnl"
	Placements []string // sent as repeated "placement"
}

// RawResponse carries the verbatim CLEN payload for pass-through endpoints.
// Body is the raw response body (snake_case, including every CLEN field).
// StatusCode is the upstream HTTP status; callers that forward to end-users
// should propagate it.
type RawResponse struct {
	StatusCode int
	Body       []byte
}

// GetLeadsRaw calls CLEN and returns the upstream body verbatim (no decoding,
// no field renaming, no drop). Non-2xx responses are NOT converted to Go
// errors — they are returned with their body so the caller can forward them
// as-is. The error return is reserved for transport failures only.
func (c *CLENLeadClient) GetLeadsRaw(ctx context.Context, cisID string, params LeadQueryParams) (*RawResponse, error) {
	if c == nil {
		return &RawResponse{StatusCode: 200, Body: []byte("[]")}, nil
	}
	if cisID == "" {
		return &RawResponse{StatusCode: 200, Body: []byte("[]")}, nil
	}
	ipID, err := strconv.ParseInt(cisID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cisID %q is not numeric: %w", cisID, err)
	}

	req := c.restClient.Client.R().
		SetContext(ctx).
		SetHeader("X-Api-Key", c.cfg.APIKey).
		SetHeader("Request-Identifier", uuid.NewString()).
		SetHeader("Request-Application-Identifier", c.cfg.AppIdentifier).
		SetHeader("Request-Datetime", time.Now().UTC().Format("2006-01-02T15:04:05.000")).
		SetQueryParam("ip_id", strconv.FormatInt(ipID, 10))
	if c.cfg.ExpireFilter != "" {
		req.SetQueryParam("exp_f", c.cfg.ExpireFilter)
	}
	if params.Channel != "" {
		req.SetQueryParam("chnl", params.Channel)
	}
	if placements := nonEmpty(params.Placements); len(placements) > 0 {
		req.SetQueryParamsFromValues(url.Values{"placement": placements})
	}
	resp, err := req.Get(c.cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("call CLEN lead: %w", err)
	}
	return &RawResponse{StatusCode: resp.StatusCode(), Body: resp.Bytes()}, nil
}

// GetLeads is the decoded/flattened view of GetLeadsRaw used by internal
// callers (e.g. resolveUserAttrs). It returns the list of leads for the
// requested cisID, dropping the customer-group wrapper.
func (c *CLENLeadClient) GetLeads(ctx context.Context, cisID string, params LeadQueryParams) ([]entity.Lead, error) {
	if c == nil {
		return nil, nil
	}
	if cisID == "" {
		return nil, nil
	}
	ipID, err := strconv.ParseInt(cisID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cisID %q is not numeric: %w", cisID, err)
	}

	raw, err := c.GetLeadsRaw(ctx, cisID, params)
	if err != nil {
		return nil, err
	}
	if raw.StatusCode >= 400 {
		snippet := string(raw.Body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("CLEN lead %d: %s", raw.StatusCode, snippet)
	}

	var body []clenLeadGroup
	if len(raw.Body) > 0 {
		if err := json.Unmarshal(raw.Body, &body); err != nil {
			return nil, fmt.Errorf("decode CLEN lead response: %w", err)
		}
	}

	for _, g := range body {
		if g.IPID != ipID {
			continue
		}
		if len(g.Leads) == 0 {
			return nil, nil
		}
		out := make([]entity.Lead, 0, len(g.Leads))
		for _, l := range g.Leads {
			out = append(out, entity.Lead{
				LeadID:           l.LeadID,
				LeadType:         l.LeadType,
				LeadStatus:       l.LeadStatus,
				ContentID:        l.ContentID,
				CSVMCampaignCode: l.CSVMCampaignCode,
				RMSTCampaignCode: l.RMSTCampaignCode,
				CampaignName:     l.CampaignName,
				ExpireDate:       l.ExpireDate,
				AssignedDate:     l.AssignedDate,
				SLAStartDate:     l.SLAStartDate,
				SLAEndDate:       l.SLAEndDate,
				ShowSLACountDown: l.ShowSLACountDown,
				FinalScore:       l.FinalScore,
				Placements:       l.Placement,
			})
		}
		return out, nil
	}
	return nil, nil
}
