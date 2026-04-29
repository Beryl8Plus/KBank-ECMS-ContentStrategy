package entity

// Lead is the subset of CLEN "Get Lead Information by Customer Level" fields
// the delivery service injects into user attributes for rule evaluation.
// Unmapped CLEN fields are intentionally dropped here to avoid locking the
// response shape before the rule engine actually conditions on them.
type Lead struct {
	LeadID           string   `json:"leadId"`
	LeadType         string   `json:"leadType"`
	LeadStatus       string   `json:"leadStatus"`
	ContentID        string   `json:"contentId"`
	CSVMCampaignCode string   `json:"csvmCampaignCode,omitempty"`
	RMSTCampaignCode string   `json:"rmstCampaignCode,omitempty"`
	CampaignName     string   `json:"campaignName,omitempty"`
	ExpireDate       string   `json:"expireDate,omitempty"`
	AssignedDate     string   `json:"assignedDate,omitempty"`
	SLAStartDate     string   `json:"slaStartDate,omitempty"`
	SLAEndDate       string   `json:"slaEndDate,omitempty"`
	ShowSLACountDown bool     `json:"showSlaCountDown"`
	FinalScore       string   `json:"finalScore,omitempty"`
	Placements       []string `json:"placements,omitempty"`
}
