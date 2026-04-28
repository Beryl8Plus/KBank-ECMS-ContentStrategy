package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/pubsub"
	"kbank-ecms/pkg/ctxconsts"
)

// Sentinel errors mapped to HTTP status codes in the handler.
var (
	ErrWizardNotFound   = errors.New("not found")
	ErrWizardConflict   = errors.New("conflict")
	ErrWizardValidation = errors.New("validation error")
)

const (
	maxConditionDepth        = 3
	maxConditionsPerGroup    = 10
	maxSchedulesPerPlacement = 3
	defaultTimezone          = "Asia/Bangkok"
)

// DecisionRuleWizardService handles business logic for the 4-step creation wizard.
type DecisionRuleWizardService struct {
	repo          domainrepo.DecisionRuleWizardRepository
	attrRepo      domainrepo.AttributeRepository
	placementRepo domainrepo.PlacementRepository
	validator     *AttributeValidatorService
}

// NewDecisionRuleWizardService creates a new DecisionRuleWizardService.
// publisher may be nil; cache invalidation pings are then suppressed and
// delivery pods rely on TTL expiry instead.
func NewDecisionRuleWizardService(
	repo domainrepo.DecisionRuleWizardRepository,
	attrRepo domainrepo.AttributeRepository,
	placementRepo domainrepo.PlacementRepository,
	publisher *pubsub.Publisher,
	validator *AttributeValidatorService,
) *DecisionRuleWizardService {
	return &DecisionRuleWizardService{repo: repo, attrRepo: attrRepo, placementRepo: placementRepo, validator: validator}
}

// ── Step 1 ───────────────────────────────────────────────────────────────────

// SaveStep1 creates a draft DecisionRule with its condition tree.
func (s *DecisionRuleWizardService) SaveStep1(ctx context.Context, req dto.WizardStep1Request) (*dto.WizardStep1Response, error) {
	if !req.Type.IsValid() {
		return nil, fmt.Errorf("%w: type must be one of MASS, AUDIENCE, SALES_TARGET, NON_SALES", ErrWizardValidation)
	}
	if !req.EvaluateType.IsValid() {
		return nil, fmt.Errorf("%w: evaluateType must be one of SCORING, SEGMENT, ELIGIBLE", ErrWizardValidation)
	}
	if req.Type.IsCampaign() && strings.TrimSpace(req.CampaignCode) == "" {
		return nil, fmt.Errorf("%w: campaignCode is required for AUDIENCE and SALES_TARGET types", ErrWizardValidation)
	}

	if err := validateConditionTree(req.Conditions, 1); err != nil {
		return nil, err
	}
	if err := validateNoDuplicateAttributes(req.Conditions); err != nil {
		return nil, err
	}
	if err := s.validateAttributeIDs(ctx, req.Conditions); err != nil {
		return nil, err
	}

	drID := uuid.New()
	dr := &entity.DecisionRule{
		BaseModel:           entity.BaseModel{ID: drID},
		DecisionRuleRunning: generateDecisionRuleID(),
		Name:                req.Name,
		Type:                req.Type,
		EvaluateType:        req.EvaluateType,
		ContentPath:         req.ContentPath,
		CampaignCode:        req.CampaignCode,
		Score:               req.Score,
		Status:              enums.DecisionRuleStatusDraft,
		SubStatus:           enums.DecisionRuleSubStatusNA,
	}

	conditions := flattenConditions(drID, req.Conditions, nil)

	if err := s.repo.SaveStep1(ctx, dr, conditions); err != nil {
		return nil, fmt.Errorf("saving step 1: %w", err)
	}

	return &dto.WizardStep1Response{
		ID:                  dr.ID,
		DecisionRuleRunning: dr.DecisionRuleRunning,
		Status:              dr.Status,
		CreatedAt:           dr.CreatedAt,
	}, nil
}

// GetConditions returns the condition tree for the edit-mode view (Step 1 read).
func (s *DecisionRuleWizardService) GetConditions(ctx context.Context, id uuid.UUID) (*dto.WizardConditionsResponse, error) {
	dr, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr == nil {
		return nil, fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}

	flat, err := s.repo.FindTemplateConditions(ctx, id)
	if err != nil {
		return nil, err
	}

	return &dto.WizardConditionsResponse{
		ID:                  dr.ID,
		DecisionRuleRunning: dr.DecisionRuleRunning,
		Type:                dr.Type,
		EvaluateType:        dr.EvaluateType,
		Name:                dr.Name,
		ContentPath:         dr.ContentPath,
		CampaignCode:        dr.CampaignCode,
		Score:               dr.Score,
		Status:              dr.Status,
		SubStatus:           dr.SubStatus,
		Conditions:          buildConditionTree(flat, nil),
	}, nil
}

// UpdateStep1 edits an existing draft DecisionRule: updates header fields,
// upserts the condition tree, and cascade-deletes Step 2 data for removed conditions.
func (s *DecisionRuleWizardService) UpdateStep1(ctx context.Context, id uuid.UUID, req dto.WizardStep1Request) (*dto.WizardEditStep1Response, error) {
	if !req.Type.IsValid() {
		return nil, fmt.Errorf("%w: type must be one of MASS, AUDIENCE, SALES_TARGET, NON_SALES", ErrWizardValidation)
	}
	if !req.EvaluateType.IsValid() {
		return nil, fmt.Errorf("%w: evaluateType must be one of SCORING, SEGMENT, ELIGIBLE", ErrWizardValidation)
	}
	if req.Type.IsCampaign() && strings.TrimSpace(req.CampaignCode) == "" {
		return nil, fmt.Errorf("%w: campaignCode is required for AUDIENCE and SALES_TARGET types", ErrWizardValidation)
	}

	dr, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr == nil {
		return nil, fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}

	if err := validateConditionTree(req.Conditions, 1); err != nil {
		return nil, err
	}
	// Option A: each attribute must appear at most once across all conditions.
	if err := validateNoDuplicateAttributes(req.Conditions); err != nil {
		return nil, err
	}
	if err := s.validateAttributeIDs(ctx, req.Conditions); err != nil {
		return nil, err
	}

	// Compute which existing conditions are being removed.
	existingConds, err := s.repo.FindTemplateConditions(ctx, id)
	if err != nil {
		return nil, err
	}
	incomingIDs := collectConditionIDs(req.Conditions)
	var toDeleteConditionIDs []uuid.UUID
	for _, c := range existingConds {
		if !incomingIDs[c.ID] {
			toDeleteConditionIDs = append(toDeleteConditionIDs, c.ID)
		}
	}

	drUpdate := &entity.DecisionRule{
		Name:         req.Name,
		Type:         req.Type,
		EvaluateType: req.EvaluateType,
		ContentPath:  req.ContentPath,
		CampaignCode: req.CampaignCode,
		Score:        req.Score,
	}
	toUpsert := flattenConditionsForEdit(id, req.Conditions, nil)

	affectedRuleIDs, err := s.repo.UpdateStep1(ctx, id, drUpdate, toUpsert, toDeleteConditionIDs)
	if err != nil {
		return nil, fmt.Errorf("updating step 1: %w", err)
	}

	resp := &dto.WizardEditStep1Response{
		ID:                  id,
		DecisionRuleRunning: dr.DecisionRuleRunning,
		Status:              dr.Status,
		UpdatedAt:           time.Now(),
	}
	if len(affectedRuleIDs) > 0 {
		resp.CascadeEffect = &dto.CascadeEffectResponse{
			DeletedRulesCount: len(affectedRuleIDs),
			AffectedRuleIDs:   affectedRuleIDs,
		}
	}
	return resp, nil
}

// ── Step 2 ───────────────────────────────────────────────────────────────────

// GetRuleSets returns columns (from template conditions) and existing rule sets.
func (s *DecisionRuleWizardService) GetRuleSets(ctx context.Context, id uuid.UUID) (*dto.WizardRuleSetsResponse, error) {
	dr, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr == nil {
		return nil, fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}

	flat, err := s.repo.FindTemplateConditions(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build columns: only leaf conditions (attribute_id IS NOT NULL).
	var columns []dto.RuleColumnResponse
	for _, c := range flat {
		col := dto.RuleColumnResponse{
			ConditionID:       c.ID,
			AttributeID:       c.AttributeID,
			AttributeName:     c.Attribute.DisplayName,
			AttributeIsActive: c.Attribute.IsActive,
			LogicalOperator:   enums.LogicalOperator(c.LogicalOperator),
			DataType:          c.Attribute.DataType,
		}
		columns = append(columns, col)
	}

	rules, err := s.repo.FindRulesByDecisionRuleID(ctx, id)
	if err != nil {
		return nil, err
	}

	var ruleIDs []uuid.UUID
	for _, r := range rules {
		ruleIDs = append(ruleIDs, r.ID)
	}

	attrs, err := s.repo.FindRuleAttributesByRuleIDs(ctx, ruleIDs)
	if err != nil {
		return nil, err
	}

	// Index attrs by (rule_id, attribute_id) — attribute is unique per decision rule.
	type attrKey struct{ ruleID, attributeID uuid.UUID }
	attrIndex := make(map[attrKey]*entity.RuleAttribute, len(attrs))
	for _, a := range attrs {
		attrIndex[attrKey{a.RuleID, a.AttributeID}] = a
	}

	ruleSets := make([]dto.RuleSetResponse, 0, len(rules))
	for _, r := range rules {
		score := &r.Score
		values := make([]dto.RuleValueResponse, 0, len(columns))
		for _, col := range columns {
			var val *string
			if a, ok := attrIndex[attrKey{r.ID, col.AttributeID}]; ok {
				if len(a.Value) > 0 {
					var raw interface{}
					if err := json.Unmarshal(a.Value, &raw); err == nil {
						s := fmt.Sprintf("%v", raw)
						val = &s
					}
				}
			}
			values = append(values, dto.RuleValueResponse{ConditionID: col.ConditionID, Value: val})
		}
		ruleSets = append(ruleSets, dto.RuleSetResponse{
			RuleID:        r.ID,
			OrderNo:       r.OrderNo,
			Score:         score,
			VariationName: r.VariationName,
			Values:        values,
		})
	}

	return &dto.WizardRuleSetsResponse{
		ID:       id,
		Columns:  columns,
		RuleSets: ruleSets,
	}, nil
}

// SaveStep2 upserts rule sets and their attribute values.
func (s *DecisionRuleWizardService) SaveStep2(ctx context.Context, id uuid.UUID, req dto.WizardStep2Request) (*dto.WizardStep2Response, error) {
	dr, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr == nil {
		return nil, fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}

	// Validate orderNo uniqueness within request.
	seen := make(map[int]bool)
	for _, rs := range req.RuleSets {
		if seen[rs.OrderNo] {
			return nil, fmt.Errorf("%w: duplicate orderNo %d in ruleSets", ErrWizardValidation, rs.OrderNo)
		}
		seen[rs.OrderNo] = true
	}

	// Collect and validate condition IDs belong to this decision rule.
	conditionIDSet := make(map[uuid.UUID]bool)
	for _, rs := range req.RuleSets {
		for _, c := range rs.Conditions {
			conditionIDSet[c.ConditionID] = true
		}
	}
	conditionIDs := make([]uuid.UUID, 0, len(conditionIDSet))
	for id := range conditionIDSet {
		conditionIDs = append(conditionIDs, id)
	}

	existingConds, err := s.repo.FindConditionsByIDs(ctx, conditionIDs)
	if err != nil {
		return nil, err
	}
	condMap := make(map[uuid.UUID]*entity.RuleCondition, len(existingConds))
	for _, c := range existingConds {
		condMap[c.ID] = c
	}
	for _, cid := range conditionIDs {
		c, ok := condMap[cid]
		if !ok {
			return nil, fmt.Errorf("%w: condition %s not found", ErrWizardNotFound, cid)
		}
		if c.DecisionRuleID != dr.ID {
			return nil, fmt.Errorf("%w: condition %s does not belong to this decision rule", ErrWizardValidation, cid)
		}
	}

	// Validate existing rule IDs and compute which rules to delete.
	existingRules, err := s.repo.FindRulesByDecisionRuleID(ctx, id)
	if err != nil {
		return nil, err
	}
	incomingRuleIDs := make(map[uuid.UUID]bool)
	for _, rs := range req.RuleSets {
		if rs.RuleID != nil {
			incomingRuleIDs[*rs.RuleID] = true
		}
	}
	var toDeleteRuleIDs []uuid.UUID
	for _, r := range existingRules {
		if !incomingRuleIDs[r.ID] {
			toDeleteRuleIDs = append(toDeleteRuleIDs, r.ID)
		}
	}

	for _, rs := range req.RuleSets {
		if rs.RuleID != nil {
			existing, err := s.repo.FindRuleByID(ctx, *rs.RuleID)
			if err != nil {
				return nil, err
			}
			if existing == nil {
				return nil, fmt.Errorf("%w: rule %s not found", ErrWizardNotFound, *rs.RuleID)
			}
		}
	}

	var rules []*entity.Rule
	var allAttrs []*entity.RuleAttribute

	for _, rs := range req.RuleSets {
		rule := &entity.Rule{
			DecisionRuleID: id,
			VariationName:  rs.VariationName,
			OrderNo:        rs.OrderNo,
		}
		if rs.Score != nil {
			rule.Score = *rs.Score
		}
		if rs.RuleID != nil {
			rule.ID = *rs.RuleID
		} else {
			rule.ID = uuid.New()
		}
		rules = append(rules, rule)

		for _, cv := range rs.Conditions {
			if cv.Value == nil {
				continue // only store non-null values
			}
			cond := condMap[cv.ConditionID]
			raw, _ := json.Marshal(cv.Value)
			attr := &entity.RuleAttribute{
				BaseModel:   entity.BaseModel{ID: uuid.New()},
				RuleID:      rule.ID,
				AttributeID: cond.AttributeID,
				Value:       raw,
			}
			allAttrs = append(allAttrs, attr)
		}
	}

	// Validate that every chosen attribute value is still in its allowed options set.
	valInputs := make([]RuleAttributeInput, 0, len(allAttrs))
	for _, a := range allAttrs {
		valInputs = append(valInputs, RuleAttributeInput{
			RuleAttributeID: a.ID,
			AttributeID:     a.AttributeID,
			ValueJSON:       a.Value,
		})
	}
	if valueErrs, err := s.validator.ValidateRuleAttributeValues(ctx, valInputs); err != nil {
		return nil, fmt.Errorf("validating rule attribute values: %w", err)
	} else if len(valueErrs) > 0 {
		return nil, fmt.Errorf("%w: attribute value no longer in allowed options: %s", ErrWizardValidation, valueErrs[0].Error())
	}

	if err := s.repo.SaveStep2(ctx, rules, allAttrs, toDeleteRuleIDs); err != nil {
		return nil, fmt.Errorf("saving step 2: %w", err)
	}

	return &dto.WizardStep2Response{
		ID:            id,
		SavedRuleSets: len(rules),
		UpdatedAt:     time.Now(),
	}, nil
}

// ── Step 3 ───────────────────────────────────────────────────────────────────

// SaveStep3 attaches schedules to the decision rule and sets its status to ACTIVE.
func (s *DecisionRuleWizardService) SaveStep3(ctx context.Context, id uuid.UUID, req dto.WizardStep3Request) (*dto.WizardStep3Response, error) {
	dr, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr == nil {
		return nil, fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}

	// Validate no duplicate placementIds within the request.
	placementSeen := make(map[uuid.UUID]bool)
	for _, sc := range req.Schedules {
		if placementSeen[sc.PlacementID] {
			return nil, fmt.Errorf("%w: duplicate placementId %s in schedules", ErrWizardValidation, sc.PlacementID)
		}
		placementSeen[sc.PlacementID] = true
	}

	var schedules []*entity.Schedule
	var placementDetails []*entity.Placement

	for _, sc := range req.Schedules {
		if !sc.RecurrenceType.IsValid() {
			return nil, fmt.Errorf("%w: invalid recurrenceType %s", ErrWizardValidation, sc.RecurrenceType)
		}
		if !sc.EndDate.After(sc.StartDate) {
			return nil, fmt.Errorf("%w: startDate must be before endDate for placement %s", ErrWizardValidation, sc.PlacementID)
		}
		if sc.EndDate.Before(time.Now()) {
			return nil, fmt.Errorf("%w: endDate must be a future date for placement %s", ErrWizardValidation, sc.PlacementID)
		}
		if !sc.AllDay {
			if sc.TimeOfDayStart == nil || sc.TimeOfDayEnd == nil {
				return nil, fmt.Errorf("%w: timeOfDayStart and timeOfDayEnd required when allDay is false for placement %s", ErrWizardValidation, sc.PlacementID)
			}
		}
		if sc.RecurrenceType == enums.RecurrenceTypeRRule && sc.RecurrenceRule == nil {
			return nil, fmt.Errorf("%w: recurrenceRule is required when recurrenceType is RRULE", ErrWizardValidation)
		}
		if sc.RecurrenceType == enums.RecurrenceTypeCalendar && sc.CalendarID == nil {
			return nil, fmt.Errorf("%w: calendarId is required when recurrenceType is CALENDAR", ErrWizardValidation)
		}

		placement, err := s.placementRepo.GetPlacementByID(ctx, sc.PlacementID)
		if err != nil {
			return nil, err
		}
		if placement == nil {
			return nil, fmt.Errorf("%w: placement %s not found", ErrWizardNotFound, sc.PlacementID)
		}
		placementDetails = append(placementDetails, placement)

		// Count schedules for this placement from OTHER decision rules.
		// SaveStep3 repo does full replacement (deletes this DR's schedules first),
		// so only other DRs count toward the placement cap.
		count, err := s.repo.CountSchedulesByPlacementExcludingDR(ctx, sc.PlacementID, id)
		if err != nil {
			return nil, err
		}
		if count >= maxSchedulesPerPlacement {
			return nil, fmt.Errorf("%w: placement %s has reached the maximum of %d schedules", ErrWizardValidation, sc.PlacementID, maxSchedulesPerPlacement)
		}

		tz := sc.Timezone
		if tz == "" {
			tz = defaultTimezone
		}

		schedule := &entity.Schedule{
			BaseModel:      entity.BaseModel{ID: uuid.New()},
			DecisionRuleID: id,
			PlacementID:    sc.PlacementID,
			CalendarID:     sc.CalendarID,
			RecurrenceType: sc.RecurrenceType,
			RecurrenceRule: sc.RecurrenceRule,
			EffectiveFrom:  sc.StartDate.UTC(),
			EffectiveUntil: sc.EndDate.UTC(),
			AllDay:         sc.AllDay,
			TimeOfDayStart: sc.TimeOfDayStart,
			TimeOfDayEnd:   sc.TimeOfDayEnd,
			Timezone:       tz,
			IsActive:       true,
		}
		schedules = append(schedules, schedule)
	}

	if err := s.repo.SaveStep3(ctx, id, schedules); err != nil {
		return nil, fmt.Errorf("saving step 3: %w", err)
	}

	// Attach placement info to schedules for response.
	for i, sc := range schedules {
		sc.Placement = placementDetails[i]
	}

	resp := &dto.WizardStep3Response{
		ID:                  id,
		DecisionRuleRunning: dr.DecisionRuleRunning,
		Status:              dr.Status,
		Schedules:           make([]dto.WizardScheduleResponse, len(schedules)),
	}
	for i, sc := range schedules {
		resp.Schedules[i] = dto.ToWizardScheduleResponse(sc)
	}
	return resp, nil
}

// ── Step 4 ───────────────────────────────────────────────────────────────────

// ActivateStep4 validates that the decision rule has schedules then sets its status to ACTIVE.
func (s *DecisionRuleWizardService) ActivateStep4(ctx context.Context, id uuid.UUID) (*dto.WizardStep4Response, error) {
	dr, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr == nil {
		return nil, fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}

	schedules, err := s.repo.FindSchedulesByDecisionRuleID(ctx, dr.ID)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 0 {
		return nil, fmt.Errorf("%w: decision rule has no schedules, complete step 3 first", ErrWizardValidation)
	}

	// Check 1: all referenced attributes must still be active.
	flatConds, err := s.repo.FindTemplateConditions(ctx, id)
	if err != nil {
		return nil, err
	}
	var attrIDs []uuid.UUID
	for _, c := range flatConds {
		if c.AttributeID != uuid.Nil {
			attrIDs = append(attrIDs, c.AttributeID)
		}
	}
	if inactiveErrs, err := s.validator.ValidateAttributesActive(ctx, attrIDs); err != nil {
		return nil, fmt.Errorf("validating attributes: %w", err)
	} else if len(inactiveErrs) > 0 {
		return nil, fmt.Errorf("%w: attribute %s (%s) is inactive — update the rule before activating", ErrWizardValidation, inactiveErrs[0].AttributeID, inactiveErrs[0].FieldName)
	}

	// Check 2: all rule attribute values must still be in their allowed options.
	rules, err := s.repo.FindRulesByDecisionRuleID(ctx, id)
	if err != nil {
		return nil, err
	}
	var ruleIDs []uuid.UUID
	for _, r := range rules {
		ruleIDs = append(ruleIDs, r.ID)
	}
	ruleAttrs, err := s.repo.FindRuleAttributesByRuleIDs(ctx, ruleIDs)
	if err != nil {
		return nil, err
	}
	valInputs := make([]RuleAttributeInput, 0, len(ruleAttrs))
	for _, ra := range ruleAttrs {
		valInputs = append(valInputs, RuleAttributeInput{
			RuleAttributeID: ra.ID,
			AttributeID:     ra.AttributeID,
			ValueJSON:       ra.Value,
		})
	}
	if valueErrs, err := s.validator.ValidateRuleAttributeValues(ctx, valInputs); err != nil {
		return nil, fmt.Errorf("validating rule values: %w", err)
	} else if len(valueErrs) > 0 {
		return nil, fmt.Errorf("%w: rule value %q is no longer in the allowed options for attribute %s", ErrWizardValidation, valueErrs[0].ChosenValue, valueErrs[0].AttributeID)
	}

	// ActivateDecisionRule also resets SUB_STATUS → "N/A".
	if err := s.repo.ActivateDecisionRule(ctx, dr.ID); err != nil {
		return nil, err
	}

	return &dto.WizardStep4Response{
		ID:                  dr.ID,
		DecisionRuleRunning: dr.DecisionRuleRunning,
		Status:              enums.DecisionRuleStatusActive,
		Schedules:           schedules,
	}, nil
}

// GetSchedules returns the schedules for the edit-mode view (Step 3 read).
func (s *DecisionRuleWizardService) GetSchedules(ctx context.Context, id uuid.UUID) (*dto.WizardSchedulesResponse, error) {
	dr, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr == nil {
		return nil, fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}

	schedules, err := s.repo.FindSchedulesByDecisionRuleID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &dto.WizardSchedulesResponse{
		ID:                  dr.ID,
		DecisionRuleRunning: dr.DecisionRuleRunning,
		Schedules:           make([]dto.WizardScheduleResponse, len(schedules)),
	}
	for i, sc := range schedules {
		resp.Schedules[i] = dto.ToWizardScheduleResponse(sc)
	}
	return resp, nil
}

// ── List ─────────────────────────────────────────────────────────────────────

// ListDecisionRules returns a paginated list with placement summaries.
func (s *DecisionRuleWizardService) ListDecisionRules(ctx context.Context, f domainrepo.DecisionRuleListFilter) ([]*dto.DecisionRuleListItemResponse, int64, error) {
	drs, total, err := s.repo.ListDecisionRules(ctx, f)
	if err != nil {
		return nil, 0, err
	}

	if len(drs) == 0 {
		return nil, 0, nil
	}

	ids := make([]uuid.UUID, len(drs))
	for i, dr := range drs {
		ids[i] = dr.ID
	}

	schedules, err := s.repo.FindSchedulesWithPlacementsByDecisionRuleIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	// Group placements by decision rule ID.
	placementsByDR := make(map[uuid.UUID][]dto.DecisionRulePlacementResponse)
	for _, sc := range schedules {
		if sc.Placement == nil {
			continue
		}
		entry := dto.DecisionRulePlacementResponse{
			PlacementID:   sc.PlacementID,
			PlacementName: sc.Placement.PlacementName,
		}
		if sc.Placement.Channel != nil {
			entry.ChannelName = sc.Placement.Channel.ChannelName
		}
		placementsByDR[sc.DecisionRuleID] = append(placementsByDR[sc.DecisionRuleID], entry)
	}

	items := make([]*dto.DecisionRuleListItemResponse, len(drs))
	for i, dr := range drs {
		placements := placementsByDR[dr.ID]
		if placements == nil {
			placements = []dto.DecisionRulePlacementResponse{}
		}
		items[i] = &dto.DecisionRuleListItemResponse{
			ID:                  dr.ID,
			DecisionRuleRunning: dr.DecisionRuleRunning,
			Name:                dr.Name,
			Type:                dr.Type,
			EvaluateType:        dr.EvaluateType,
			CampaignCode:        dr.CampaignCode,
			Status:              dr.Status,
			SubStatus:           dr.SubStatus,
			Placements:          placements,
			CreatedBy:           toUserRef(dr.CreatedByUser, dr.BaseModel.CreatedBy),
			UpdatedBy:           toUserRef(dr.UpdatedByUser, dr.BaseModel.UpdatedBy),
			InactiveBy:          toUserRef(dr.InactiveByUser, dr.InactiveBy),
			CreatedAt:           dr.CreatedAt,
			UpdatedAt:           dr.UpdatedAt,
		}
	}
	return items, total, nil
}

func toUserRef(u *entity.User, id *uuid.UUID) *dto.UserRefResponse {
	if id == nil {
		return nil
	}
	ref := &dto.UserRefResponse{UserID: id}
	if u != nil {
		ref.NameTH = u.NameTH
		ref.NameEN = u.NameEN
	}
	return ref
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

// CloneDecisionRule performs a deep copy of an existing rule into a new DRAFT.
// Fail-fast: if the source is INACTIVE with sub-status "Missing attribute registry"
// the operation is rejected immediately before any DB work.
func (s *DecisionRuleWizardService) CloneDecisionRule(ctx context.Context, id uuid.UUID) (*dto.CloneDecisionRuleResponse, error) {
	orig, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if orig == nil {
		return nil, fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}

	// Fail-fast: broken rules cannot be cloned.
	if orig.Status == enums.DecisionRuleStatusInactive && orig.SubStatus == enums.DecisionRuleSubStatusMissing {
		return nil, fmt.Errorf("%w: cannot clone a rule with status INACTIVE and sub-status '%s'", ErrWizardValidation, enums.DecisionRuleSubStatusMissing)
	}

	flatConds, err := s.repo.FindTemplateConditions(ctx, id)
	if err != nil {
		return nil, err
	}
	rules, err := s.repo.FindRulesByDecisionRuleID(ctx, id)
	if err != nil {
		return nil, err
	}
	ruleIDs := make([]uuid.UUID, len(rules))
	for i, r := range rules {
		ruleIDs[i] = r.ID
	}
	attrs, err := s.repo.FindRuleAttributesByRuleIDs(ctx, ruleIDs)
	if err != nil {
		return nil, err
	}
	schedules, err := s.repo.FindSchedulesByDecisionRuleID(ctx, id)
	if err != nil {
		return nil, err
	}

	newDRID := uuid.New()
	newDR := &entity.DecisionRule{
		BaseModel:    entity.BaseModel{ID: newDRID},
		Name:         orig.Name,
		Type:         orig.Type,
		EvaluateType: orig.EvaluateType,
		ContentPath:  orig.ContentPath,
		CampaignCode: orig.CampaignCode,
		Score:        orig.Score,
		Status:       enums.DecisionRuleStatusDraft,
		SubStatus:    enums.DecisionRuleSubStatusNA,
		// DecisionRuleRunning intentionally empty — BeforeCreate hook generates it.
	}

	// Deep copy conditions, preserving parent-child links with remapped IDs.
	oldToNewCond := make(map[uuid.UUID]uuid.UUID, len(flatConds))
	newConds := make([]*entity.RuleCondition, len(flatConds))
	for i, c := range flatConds {
		newID := uuid.New()
		oldToNewCond[c.ID] = newID
		newConds[i] = &entity.RuleCondition{
			BaseModel:         entity.BaseModel{ID: newID},
			DecisionRuleID:    newDRID,
			Sequence:          c.Sequence,
			AttributeID:       c.AttributeID,
			LogicalOperator:   c.LogicalOperator,
			ConnectorOperator: c.ConnectorOperator,
		}
	}
	for i, c := range flatConds {
		if c.ParentRuleConditionID != nil {
			if newParentID, ok := oldToNewCond[*c.ParentRuleConditionID]; ok {
				newConds[i].ParentRuleConditionID = &newParentID
			}
		}
	}

	// Deep copy rules.
	oldToNewRule := make(map[uuid.UUID]uuid.UUID, len(rules))
	newRules := make([]*entity.Rule, len(rules))
	for i, r := range rules {
		newRuleID := uuid.New()
		oldToNewRule[r.ID] = newRuleID
		newRules[i] = &entity.Rule{
			BaseModel:      entity.BaseModel{ID: newRuleID},
			DecisionRuleID: newDRID,
			VariationName:  r.VariationName,
			Score:          r.Score,
			OrderNo:        r.OrderNo,
		}
	}

	// Deep copy rule attributes, pointing to new rule IDs.
	newAttrs := make([]*entity.RuleAttribute, len(attrs))
	for i, a := range attrs {
		newAttrs[i] = &entity.RuleAttribute{
			BaseModel:   entity.BaseModel{ID: uuid.New()},
			RuleID:      oldToNewRule[a.RuleID],
			AttributeID: a.AttributeID,
			Value:       a.Value,
		}
	}

	// Build placeholder schedules: carry over placement info only.
	// Time fields (EffectiveFrom, EffectiveUntil, TimeOfDayStart, TimeOfDayEnd) are
	// intentionally zeroed so the user must choose new dates in Step 3.
	newSchedules := make([]*entity.Schedule, len(schedules))
	for i, sc := range schedules {
		newSchedules[i] = &entity.Schedule{
			BaseModel:      entity.BaseModel{ID: uuid.New()},
			DecisionRuleID: newDRID,
			PlacementID:    sc.PlacementID,
			CalendarID:     sc.CalendarID,
			RecurrenceType: sc.RecurrenceType,
			RecurrenceRule: sc.RecurrenceRule,
			AllDay:         sc.AllDay,
			Timezone:       sc.Timezone,
			IsActive:       false,
		}
	}

	if err := s.repo.CloneDecisionRule(ctx, newDR, newConds, newRules, newAttrs, newSchedules); err != nil {
		return nil, fmt.Errorf("cloning decision rule: %w", err)
	}

	return &dto.CloneDecisionRuleResponse{
		ID:                  newDR.ID,
		DecisionRuleRunning: newDR.DecisionRuleRunning,
		Status:              newDR.Status,
		CreatedAt:           newDR.CreatedAt,
	}, nil
}

// DeactivateDecisionRule transitions an ACTIVE rule to INACTIVE.
// Returns ErrWizardValidation if the rule is not currently ACTIVE.
func (s *DecisionRuleWizardService) DeactivateDecisionRule(ctx context.Context, id uuid.UUID) (*dto.DeactivateDecisionRuleResponse, error) {
	dr, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr == nil {
		return nil, fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}
	if dr.Status != enums.DecisionRuleStatusActive {
		return nil, fmt.Errorf("%w: only ACTIVE decision rules can be deactivated, current status is %s", ErrWizardValidation, dr.Status)
	}

	inactiveBy, _ := ctxconsts.GetUserID(ctx)
	if err := s.repo.DeactivateDecisionRule(ctx, id, inactiveBy); err != nil {
		return nil, err
	}

	return &dto.DeactivateDecisionRuleResponse{
		ID:                  dr.ID,
		DecisionRuleRunning: dr.DecisionRuleRunning,
		Status:              enums.DecisionRuleStatusInactive,
		UpdatedAt:           time.Now(),
	}, nil
}

// DeleteDecisionRule soft-deletes a DRAFT or INACTIVE rule and all its child records.
// Returns ErrWizardValidation if the rule is ACTIVE.
func (s *DecisionRuleWizardService) DeleteDecisionRule(ctx context.Context, id uuid.UUID) error {
	dr, err := s.repo.FindDecisionRuleByID(ctx, id)
	if err != nil {
		return err
	}
	if dr == nil {
		return fmt.Errorf("%w: decision rule not found", ErrWizardNotFound)
	}
	if dr.Status == enums.DecisionRuleStatusActive {
		return fmt.Errorf("%w: ACTIVE decision rules cannot be deleted; deactivate first", ErrWizardValidation)
	}
	return s.repo.DeleteDecisionRule(ctx, id)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func generateDecisionRuleID() string {
	return fmt.Sprintf("RS_%s_%s", time.Now().Format("20060102"), strings.ToUpper(uuid.New().String()[:4]))
}

// validateConditionTree checks depth, group constraints, and connector rules.
func validateConditionTree(items []dto.ConditionItemRequest, depth int) error {
	if depth > maxConditionDepth {
		return fmt.Errorf("%w: conditions exceed maximum nesting depth of %d", ErrWizardValidation, maxConditionDepth)
	}
	if len(items) > maxConditionsPerGroup {
		return fmt.Errorf("%w: group exceeds maximum of %d conditions", ErrWizardValidation, maxConditionsPerGroup)
	}
	for i, item := range items {
		if !item.Type.IsValid() {
			return fmt.Errorf("%w: invalid condition type %q", ErrWizardValidation, item.Type)
		}
		isLast := i == len(items)-1
		if isLast && item.ConnectorOperator != "" {
			return fmt.Errorf("%w: last condition in array must have connectorOperator null", ErrWizardValidation)
		}
		if item.Type == enums.ConditionTypeCondition {
			if item.AttributeID == uuid.Nil {
				return fmt.Errorf("%w: attributeId is required for condition type", ErrWizardValidation)
			}
			if item.LogicalOperator == "" {
				return fmt.Errorf("%w: logicalOperator is required for condition type", ErrWizardValidation)
			}
		}
		if item.Type == enums.ConditionTypeGroup {
			if len(item.Conditions) == 0 {
				return fmt.Errorf("%w: group must contain at least one condition", ErrWizardValidation)
			}
			if err := validateConditionTree(item.Conditions, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateAttributeIDs collects all attributeIds from the tree, checks they exist,
// and rejects any attribute that has been deactivated.
func (s *DecisionRuleWizardService) validateAttributeIDs(ctx context.Context, items []dto.ConditionItemRequest) error {
	seen := make(map[uuid.UUID]bool)
	var collect func([]dto.ConditionItemRequest)
	collect = func(items []dto.ConditionItemRequest) {
		for _, item := range items {
			if item.AttributeID != uuid.Nil {
				seen[item.AttributeID] = true
			}
			collect(item.Conditions)
		}
	}
	collect(items)

	for id := range seen {
		attr, err := s.attrRepo.GetAttributeByID(ctx, id)
		if err != nil {
			return err
		}
		if attr == nil {
			return fmt.Errorf("%w: attribute %s not found", ErrWizardNotFound, id)
		}
		if !attr.IsActive {
			return fmt.Errorf("%w: attribute %s (%s) is inactive and cannot be used in new rules", ErrWizardValidation, id, attr.FieldName)
		}
	}
	return nil
}

// flattenConditions recursively flattens a condition tree into a slice with
// pre-assigned UUIDs so that parent_rule_condition_id references are correct.
func flattenConditions(decisionRuleID uuid.UUID, items []dto.ConditionItemRequest, parentID *uuid.UUID) []*entity.RuleCondition {
	result := make([]*entity.RuleCondition, 0, len(items))
	for _, item := range items {
		cond := &entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: uuid.New()},
			DecisionRuleID:        decisionRuleID,
			ParentRuleConditionID: parentID,
			Sequence:              item.Sequence,
			AttributeID:           item.AttributeID,
			LogicalOperator:       item.LogicalOperator,
			ConnectorOperator:     item.ConnectorOperator,
		}
		result = append(result, cond)
		if len(item.Conditions) > 0 {
			result = append(result, flattenConditions(decisionRuleID, item.Conditions, &cond.ID)...)
		}
	}
	return result
}

// buildConditionTree reconstructs a nested tree from a flat slice using parent_rule_condition_id.
func buildConditionTree(flat []*entity.RuleCondition, parentID *uuid.UUID) []dto.ConditionItemResponse {
	var result []dto.ConditionItemResponse
	for _, c := range flat {
		if !uuidPtrEquals(c.ParentRuleConditionID, parentID) {
			continue
		}
		condType := enums.ConditionTypeCondition
		if c.AttributeID == uuid.Nil {
			condType = enums.ConditionTypeGroup
		}
		attrName := ""
		attrActive := false
		if c.Attribute != nil {
			attrName = c.Attribute.DisplayName
			attrActive = c.Attribute.IsActive
		}
		var attrID *uuid.UUID
		if c.AttributeID != uuid.Nil {
			id := c.AttributeID
			attrID = &id
		}
		item := dto.ConditionItemResponse{
			ConditionID:       c.ID,
			Type:              condType,
			Sequence:          c.Sequence,
			AttributeID:       attrID,
			AttributeName:     attrName,
			AttributeIsActive: attrActive,
			LogicalOperator:   c.LogicalOperator,
			ConnectorOperator: c.ConnectorOperator,
		}
		if condType == enums.ConditionTypeGroup {
			item.Conditions = buildConditionTree(flat, &c.ID)
		}
		result = append(result, item)
	}
	return result
}

func uuidPtrEquals(a, b *uuid.UUID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// TotalPages computes the number of pages for pagination.
func TotalPages(total int64, limit int) int {
	return int(math.Ceil(float64(total) / float64(limit)))
}

// validateNoDuplicateAttributes rejects condition trees that reuse the same attributeId
// across any leaf nodes. This is required by Option A of the cascade-delete strategy:
// cascade operates via attributeId, so duplicates would cause over-deletion.
func validateNoDuplicateAttributes(items []dto.ConditionItemRequest) error {
	seen := make(map[uuid.UUID]bool)
	var check func([]dto.ConditionItemRequest) error
	check = func(items []dto.ConditionItemRequest) error {
		for _, item := range items {
			if item.AttributeID != uuid.Nil {
				if seen[item.AttributeID] {
					return fmt.Errorf("%w: attributeId %s appears more than once in conditions — each attribute may only be used once", ErrWizardValidation, item.AttributeID)
				}
				seen[item.AttributeID] = true
			}
			if err := check(item.Conditions); err != nil {
				return err
			}
		}
		return nil
	}
	return check(items)
}

// collectConditionIDs returns a set of all non-nil conditionIDs present in the request tree.
func collectConditionIDs(items []dto.ConditionItemRequest) map[uuid.UUID]bool {
	result := make(map[uuid.UUID]bool)
	var collect func([]dto.ConditionItemRequest)
	collect = func(items []dto.ConditionItemRequest) {
		for _, item := range items {
			if item.ConditionID != nil {
				result[*item.ConditionID] = true
			}
			collect(item.Conditions)
		}
	}
	collect(items)
	return result
}

// flattenConditionsForEdit is like flattenConditions but preserves existing IDs from the
// request (ConditionID != nil) and generates new UUIDs only for new conditions.
func flattenConditionsForEdit(decisionRuleID uuid.UUID, items []dto.ConditionItemRequest, parentID *uuid.UUID) []*entity.RuleCondition {
	result := make([]*entity.RuleCondition, 0, len(items))
	for _, item := range items {
		condID := uuid.New()
		if item.ConditionID != nil {
			condID = *item.ConditionID
		}
		cond := &entity.RuleCondition{
			BaseModel:             entity.BaseModel{ID: condID},
			DecisionRuleID:        decisionRuleID,
			ParentRuleConditionID: parentID,
			Sequence:              item.Sequence,
			AttributeID:           item.AttributeID,
			LogicalOperator:       item.LogicalOperator,
			ConnectorOperator:     item.ConnectorOperator,
		}
		result = append(result, cond)
		if len(item.Conditions) > 0 {
			result = append(result, flattenConditionsForEdit(decisionRuleID, item.Conditions, &cond.ID)...)
		}
	}
	return result
}
