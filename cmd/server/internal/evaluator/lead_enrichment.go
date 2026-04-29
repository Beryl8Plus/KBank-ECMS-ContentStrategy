package evaluator

import (
	"strconv"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
)

// expandWithLeads turns a single SALES_TARGET ContentResult into one entry per
// lead that lists placementName in its Placements set. Lead fields override
// the rule-derived ones — ContentPath, Score, Campaign.Code, StartDateTime,
// EndDateTime — because for sales-targeted rules the actual offering is
// owned by the upstream Lead system, not the rule itself.
//
// Returns nil when no leads match, signalling "no offer" (the rule's own
// entry is dropped — a SALES_TARGET rule without a matching lead means the
// customer has no targeted offering for this placement).
//
// fallbackScore is used when a lead's FinalScore is empty or non-numeric.
func expandWithLeads(entry dto.ContentResult, leads []entity.Lead, placementName string, fallbackScore float64) []dto.ContentResult {
	if len(leads) == 0 {
		return nil
	}
	out := make([]dto.ContentResult, 0)
	for _, lead := range leads {
		if !leadMatchesPlacement(lead, placementName) {
			continue
		}
		clone := entry
		clone.ContentPath = lead.ContentID
		clone.Score = parseLeadScore(lead.FinalScore, fallbackScore)
		clone.Campaign = &dto.Campaign{Code: lead.CSVMCampaignCode}
		clone.StartDateTime = lead.SLAStartDate
		clone.EndDateTime = lead.SLAEndDate
		out = append(out, clone)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func leadMatchesPlacement(lead entity.Lead, placementName string) bool {
	if placementName == "" {
		return false
	}
	for _, p := range lead.Placements {
		if p == placementName {
			return true
		}
	}
	return false
}

func parseLeadScore(raw string, fallback float64) float64 {
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}
