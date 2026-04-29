package evaluator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
)

func TestExpandWithLeads_NoLeads(t *testing.T) {
	entry := dto.ContentResult{ContentPath: "/rule/path", Score: 50}
	got := expandWithLeads(entry, nil, "wsaHomeBanner", 50)
	assert.Nil(t, got)
}

func TestExpandWithLeads_NoMatchingPlacement(t *testing.T) {
	entry := dto.ContentResult{ContentPath: "/rule/path", Score: 50}
	leads := []entity.Lead{
		{ContentID: "/lead/a", Placements: []string{"otherPlacement"}},
	}
	got := expandWithLeads(entry, leads, "wsaHomeBanner", 50)
	assert.Nil(t, got)
}

func TestExpandWithLeads_OverridesAllFields(t *testing.T) {
	entry := dto.ContentResult{
		DecisionRuleId: "rule-1",
		ContentPath:    "/rule/path",
		Score:          50,
		StartDateTime:  "2026-01-01T00:00:00Z",
		EndDateTime:    "2026-12-31T23:59:59Z",
		LogicEval:      true,
	}
	leads := []entity.Lead{
		{
			ContentID:        "/lead/lead-001.jpg",
			FinalScore:       "92.5",
			CSVMCampaignCode: "CSVM-2026-Q2",
			SLAStartDate:     "2026-04-01T00:00:00Z",
			SLAEndDate:       "2026-06-30T23:59:59Z",
			Placements:       []string{"wsaHomeBanner"},
		},
	}

	got := expandWithLeads(entry, leads, "wsaHomeBanner", entry.Score)

	if assert.Len(t, got, 1) {
		out := got[0]
		assert.Equal(t, "rule-1", out.DecisionRuleId, "rule fields preserved")
		assert.True(t, out.LogicEval)
		assert.Equal(t, "/lead/lead-001.jpg", out.ContentPath)
		assert.InDelta(t, 92.5, out.Score, 0.0001)
		if assert.NotNil(t, out.Campaign) {
			assert.Equal(t, "CSVM-2026-Q2", out.Campaign.Code)
		}
		assert.Equal(t, "2026-04-01T00:00:00Z", out.StartDateTime)
		assert.Equal(t, "2026-06-30T23:59:59Z", out.EndDateTime)
	}
}

func TestExpandWithLeads_MultipleLeadsSamePlacement(t *testing.T) {
	entry := dto.ContentResult{ContentPath: "/rule/path", Score: 50}
	leads := []entity.Lead{
		{ContentID: "/lead/a", FinalScore: "70", Placements: []string{"wsaHomeBanner"}},
		{ContentID: "/lead/b", FinalScore: "30", Placements: []string{"wsaHomeBanner"}},
		{ContentID: "/lead/c", FinalScore: "55", Placements: []string{"otherPlacement"}},
	}

	got := expandWithLeads(entry, leads, "wsaHomeBanner", 50)

	if assert.Len(t, got, 2) {
		paths := []string{got[0].ContentPath, got[1].ContentPath}
		assert.Contains(t, paths, "/lead/a")
		assert.Contains(t, paths, "/lead/b")
		assert.NotContains(t, paths, "/lead/c")
	}
}

func TestExpandWithLeads_InvalidScoreFallsBack(t *testing.T) {
	entry := dto.ContentResult{ContentPath: "/rule/path", Score: 50}
	leads := []entity.Lead{
		{ContentID: "/lead/x", FinalScore: "not-a-number", Placements: []string{"wsaHomeBanner"}},
	}
	got := expandWithLeads(entry, leads, "wsaHomeBanner", 42)
	if assert.Len(t, got, 1) {
		assert.InDelta(t, 42.0, got[0].Score, 0.0001, "fallback score used when FinalScore is non-numeric")
	}
}

func TestExpandWithLeads_EmptyScoreFallsBack(t *testing.T) {
	entry := dto.ContentResult{ContentPath: "/rule/path", Score: 50}
	leads := []entity.Lead{
		{ContentID: "/lead/x", FinalScore: "", Placements: []string{"wsaHomeBanner"}},
	}
	got := expandWithLeads(entry, leads, "wsaHomeBanner", 17.5)
	if assert.Len(t, got, 1) {
		assert.InDelta(t, 17.5, got[0].Score, 0.0001)
	}
}

func TestLeadMatchesPlacement(t *testing.T) {
	cases := []struct {
		name      string
		lead      entity.Lead
		placement string
		want      bool
	}{
		{"empty placement", entity.Lead{Placements: []string{"a"}}, "", false},
		{"match", entity.Lead{Placements: []string{"a", "b"}}, "b", true},
		{"no match", entity.Lead{Placements: []string{"a"}}, "b", false},
		{"empty placements", entity.Lead{Placements: nil}, "a", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, leadMatchesPlacement(tc.lead, tc.placement))
		})
	}
}
