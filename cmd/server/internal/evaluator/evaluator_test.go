package evaluator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

func TestNewLocalEvaluator(t *testing.T) {
	t.Parallel()
	if NewLocalEvaluator() == nil {
		t.Fatal("NewLocalEvaluator returned nil")
	}
}

// massSchedule builds a schedule + a MASS rule with no conditions/variations.
func massSchedule(contentPath string, score float64) *entity.Schedule {
	rule := entity.DecisionRule{
		BaseModel:   entity.BaseModel{ID: uuid.New()},
		Type:        enums.DecisionTypeMass,
		Score:       score,
		ContentPath: contentPath,
	}
	return &entity.Schedule{
		BaseModel:      entity.BaseModel{ID: uuid.New()},
		DecisionRuleID: rule.ID,
		DecisionRule:   &rule,
		EffectiveFrom:  time.Now(),
		EffectiveUntil: time.Now().Add(24 * time.Hour),
	}
}

// audienceSchedule builds an AUDIENCE rule with one condition that compares
// an attribute against a string value.
func audienceSchedule(contentPath string, attrID uuid.UUID, expected string, op enums.LogicalOperator) *entity.Schedule {
	rule := entity.DecisionRule{
		BaseModel:   entity.BaseModel{ID: uuid.New()},
		Type:        enums.DecisionTypeAudience,
		Score:       2.0,
		ContentPath: contentPath,
		RuleConditions: []entity.RuleCondition{
			{
				BaseModel:         entity.BaseModel{ID: uuid.New()},
				AttributeID:       attrID,
				Sequence:          1,
				LogicalOperator:   op,
				ConnectorOperator: connectorPtr(enums.ConnectorOperatorAND),
				Attribute:         &entity.Attribute{DataType: enums.AttributeDataTypeText},
			},
		},
		Rules: []entity.Rule{{
			BaseModel:     entity.BaseModel{ID: uuid.New()},
			VariationName: "v1",
			Score:         5,
			OrderNo:       1,
			RuleAttributes: []entity.RuleAttribute{
				{AttributeID: attrID, Value: datatypes.JSON(expected)},
			},
		}},
	}
	return &entity.Schedule{
		BaseModel:      entity.BaseModel{ID: uuid.New()},
		DecisionRuleID: rule.ID,
		DecisionRule:   &rule,
		EffectiveFrom:  time.Now(),
		EffectiveUntil: time.Now().Add(24 * time.Hour),
	}
}

func TestEvaluate_MassRulePassesUnconditionally(t *testing.T) {
	t.Parallel()
	e := NewLocalEvaluator()
	got, err := e.Evaluate(context.Background(), "hero", []*entity.Schedule{
		massSchedule("/a", 1.0),
	}, map[string]json.RawMessage{}, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	if got[0].ContentPath != "/a" {
		t.Errorf("ContentPath mismatch: %q", got[0].ContentPath)
	}
}

func TestEvaluate_AudienceRuleMatches(t *testing.T) {
	t.Parallel()
	attrID := uuid.New()
	e := NewLocalEvaluator()

	got, err := e.Evaluate(context.Background(), "hero",
		[]*entity.Schedule{audienceSchedule("/match", attrID, `"gold"`, enums.LogicalOperatorEQ)},
		map[string]json.RawMessage{attrID.String(): json.RawMessage(`"gold"`)},
		nil,
	)
	require.NoError(t, err)
	require.Len(t, got, 1)
	if got[0].ContentPath != "/match" {
		t.Errorf("ContentPath = %q", got[0].ContentPath)
	}
}

func TestEvaluate_NoMatchReturnsEmpty(t *testing.T) {
	t.Parallel()
	attrID := uuid.New()
	e := NewLocalEvaluator()

	got, _ := e.Evaluate(context.Background(), "hero",
		[]*entity.Schedule{audienceSchedule("/no-match", attrID, `"gold"`, enums.LogicalOperatorEQ)},
		map[string]json.RawMessage{attrID.String(): json.RawMessage(`"silver"`)},
		nil,
	)
	if len(got) != 0 {
		t.Errorf("expected empty results, got %d", len(got))
	}
}

func TestEvaluate_SkipsScheduleWithNilDecisionRule(t *testing.T) {
	t.Parallel()
	e := NewLocalEvaluator()
	bad := &entity.Schedule{BaseModel: entity.BaseModel{ID: uuid.New()}}
	got, err := e.Evaluate(context.Background(), "hero",
		[]*entity.Schedule{bad, massSchedule("/p", 1.0)},
		map[string]json.RawMessage{}, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestEvaluate_SortsByScoreDesc(t *testing.T) {
	t.Parallel()
	e := NewLocalEvaluator()
	got, _ := e.Evaluate(context.Background(), "hero",
		[]*entity.Schedule{massSchedule("/low", 1.0), massSchedule("/high", 5.0)},
		map[string]json.RawMessage{}, nil)
	require.Len(t, got, 2)
	if got[0].ContentPath != "/high" || got[0].Score != 5.0 {
		t.Errorf("expected /high first with score=5, got %v", got)
	}
}

func TestEvaluate_SalesTargetExpandsWithLeads(t *testing.T) {
	t.Parallel()
	rule := entity.DecisionRule{
		BaseModel:    entity.BaseModel{ID: uuid.New()},
		Type:         enums.DecisionTypeSalesTarget,
		Score:        3.0,
		ContentPath:  "/sales",
		CampaignCode: "CAMP",
	}
	sched := &entity.Schedule{
		BaseModel:      entity.BaseModel{ID: uuid.New()},
		DecisionRuleID: rule.ID,
		DecisionRule:   &rule,
		EffectiveFrom:  time.Now(),
		EffectiveUntil: time.Now().Add(24 * time.Hour),
	}
	leads := []entity.Lead{{LeadID: "L1", CSVMCampaignCode: "CAMP", Placements: []string{"hero"}}}
	e := NewLocalEvaluator()
	got, err := e.Evaluate(context.Background(), "hero", []*entity.Schedule{sched},
		map[string]json.RawMessage{}, leads)
	require.NoError(t, err)
	if len(got) == 0 {
		t.Error("expected sales-target entry when lead matches")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BuildPlacementLogicEntries / buildLogicEntry
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildPlacementLogicEntries_NoVariations(t *testing.T) {
	t.Parallel()
	rule := entity.DecisionRule{
		BaseModel:   entity.BaseModel{ID: uuid.New()},
		Type:        enums.DecisionTypeMass,
		Score:       7,
		ContentPath: "/x",
	}
	sched := &entity.Schedule{
		BaseModel:      entity.BaseModel{ID: uuid.New()},
		DecisionRuleID: rule.ID,
		EffectiveFrom:  time.Now(),
		EffectiveUntil: time.Now().Add(time.Hour),
	}
	got := BuildPlacementLogicEntries(rule, sched, "hero", nil)
	require.Len(t, got, 1)
	if got[0].Score != 7 || got[0].ContentPath != "/x" {
		t.Errorf("entry mismatch: %+v", got[0])
	}
}

func TestBuildPlacementLogicEntries_WithVariations(t *testing.T) {
	t.Parallel()
	attrID := uuid.New()
	rule := entity.DecisionRule{
		BaseModel:   entity.BaseModel{ID: uuid.New()},
		Type:        enums.DecisionTypeAudience,
		ContentPath: "/v",
		RuleConditions: []entity.RuleCondition{{
			BaseModel:       entity.BaseModel{ID: uuid.New()},
			AttributeID:     attrID,
			LogicalOperator: enums.LogicalOperatorEQ,
			Attribute:       &entity.Attribute{DataType: enums.AttributeDataTypeText},
		}},
		Rules: []entity.Rule{
			{BaseModel: entity.BaseModel{ID: uuid.New()}, VariationName: "v1", Score: 5, OrderNo: 1,
				RuleAttributes: []entity.RuleAttribute{{AttributeID: attrID, Value: datatypes.JSON(`"a"`)}}},
			{BaseModel: entity.BaseModel{ID: uuid.New()}, VariationName: "v2", Score: 3, OrderNo: 2,
				RuleAttributes: []entity.RuleAttribute{{AttributeID: attrID, Value: datatypes.JSON(`"b"`)}}},
		},
	}
	sched := &entity.Schedule{
		BaseModel:      entity.BaseModel{ID: uuid.New()},
		DecisionRuleID: rule.ID,
		EffectiveFrom:  time.Now(),
		EffectiveUntil: time.Now().Add(time.Hour),
	}
	camp := &dto.Campaign{Code: "C1"}
	got := BuildPlacementLogicEntries(rule, sched, "hero", camp)
	require.Len(t, got, 2)
	for _, entry := range got {
		if entry.LogicHash == "" || entry.LogicExpr == "" {
			t.Error("LogicHash and LogicExpr should be populated")
		}
		if entry.Campaign == nil || entry.Campaign.Code != "C1" {
			t.Error("Campaign should round-trip")
		}
	}
}
