package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/pubsub"
)

// stubWizardRepo embeds the domain interface so unimplemented methods compile;
// the test only exercises the three methods ActivateStep4 calls.
type stubWizardRepo struct {
	domainrepo.DecisionRuleWizardRepository
	rule           *entity.DecisionRule
	schedules      []*entity.Schedule
	activateCalls  int
	activateReturn error
}

func (s *stubWizardRepo) FindDecisionRuleByID(_ context.Context, _ uuid.UUID) (*entity.DecisionRule, error) {
	return s.rule, nil
}

func (s *stubWizardRepo) FindSchedulesByDecisionRuleID(_ context.Context, _ uuid.UUID) ([]*entity.Schedule, error) {
	return s.schedules, nil
}

func (s *stubWizardRepo) ActivateDecisionRule(_ context.Context, _ uuid.UUID) error {
	s.activateCalls++
	return s.activateReturn
}

// recordingCacheRepo captures Publish payloads. Subscribe and other interface
// methods are not exercised by this test.
type recordingCacheRepo struct {
	domainrepo.RedisCacheRepository

	mu         sync.Mutex
	channels   []string
	payloads   []string
	publishErr error
}

func (r *recordingCacheRepo) Publish(_ context.Context, channel, payload string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels = append(r.channels, channel)
	r.payloads = append(r.payloads, payload)
	return r.publishErr
}

func (r *recordingCacheRepo) snapshot() ([]string, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out1 := append([]string(nil), r.channels...)
	out2 := append([]string(nil), r.payloads...)
	return out1, out2
}

func scheduleForPlacement(placementName string) *entity.Schedule {
	return &entity.Schedule{
		BaseModel:      entity.BaseModel{ID: uuid.New()},
		DecisionRuleID: uuid.New(),
		PlacementID:    uuid.New(),
		Placement:      &entity.Placement{PlacementName: placementName},
		EffectiveFrom:  time.Now(),
		EffectiveUntil: time.Now().Add(24 * time.Hour),
	}
}

func TestActivateStep4_PublishesOnePingPerPlacement(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()
	repo := &stubWizardRepo{
		rule: &entity.DecisionRule{BaseModel: entity.BaseModel{ID: ruleID}},
		// Two distinct placements + one duplicate of the first → expect 2 pings.
		schedules: []*entity.Schedule{
			scheduleForPlacement("home_hero"),
			scheduleForPlacement("checkout_banner"),
			scheduleForPlacement("home_hero"),
		},
	}
	cache := &recordingCacheRepo{}
	publisher := pubsub.NewPublisher(cache)

	svc := NewDecisionRuleWizardService(repo, nil, nil, publisher)

	resp, err := svc.ActivateStep4(context.Background(), ruleID)
	if err != nil {
		t.Fatalf("ActivateStep4 returned error: %v", err)
	}
	if resp == nil || resp.Status != enums.DecisionRuleStatusActive {
		t.Fatalf("expected ACTIVE response, got %+v", resp)
	}
	if repo.activateCalls != 1 {
		t.Fatalf("expected ActivateDecisionRule called once, got %d", repo.activateCalls)
	}

	channels, payloads := cache.snapshot()
	if len(channels) != 2 {
		t.Fatalf("expected 2 publishes (deduped), got %d (%v)", len(channels), channels)
	}

	gotPlacements := map[string]bool{}
	for i, ch := range channels {
		if ch != pubsub.ChannelCMSSyncPing {
			t.Fatalf("publish[%d] on wrong channel %q", i, ch)
		}
		var msg pubsub.SyncPingMessage
		if jsonErr := json.Unmarshal([]byte(payloads[i]), &msg); jsonErr != nil {
			t.Fatalf("publish[%d] payload not valid JSON: %v", i, jsonErr)
		}
		if msg.DecisionRuleID != ruleID.String() {
			t.Errorf("publish[%d] decision_rule_id = %q, want %q", i, msg.DecisionRuleID, ruleID.String())
		}
		if msg.VersionHash != "" {
			t.Errorf("publish[%d] version_hash = %q, want empty (force refresh)", i, msg.VersionHash)
		}
		gotPlacements[msg.PlacementName] = true
	}
	for _, want := range []string{"home_hero", "checkout_banner"} {
		if !gotPlacements[want] {
			t.Errorf("missing ping for placement %q (got %v)", want, gotPlacements)
		}
	}
}

func TestActivateStep4_NilPublisherIsNoOp(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()
	repo := &stubWizardRepo{
		rule:      &entity.DecisionRule{BaseModel: entity.BaseModel{ID: ruleID}},
		schedules: []*entity.Schedule{scheduleForPlacement("home_hero")},
	}

	svc := NewDecisionRuleWizardService(repo, nil, nil, nil)

	if _, err := svc.ActivateStep4(context.Background(), ruleID); err != nil {
		t.Fatalf("ActivateStep4 with nil publisher returned error: %v", err)
	}
	if repo.activateCalls != 1 {
		t.Fatalf("expected ActivateDecisionRule called once, got %d", repo.activateCalls)
	}
}

func TestActivateStep4_PublishErrorDoesNotFailActivation(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()
	repo := &stubWizardRepo{
		rule:      &entity.DecisionRule{BaseModel: entity.BaseModel{ID: ruleID}},
		schedules: []*entity.Schedule{scheduleForPlacement("home_hero")},
	}
	cache := &recordingCacheRepo{publishErr: errFakePublish}
	svc := NewDecisionRuleWizardService(repo, nil, nil, pubsub.NewPublisher(cache))

	resp, err := svc.ActivateStep4(context.Background(), ruleID)
	if err != nil {
		t.Fatalf("ActivateStep4 should not surface publish errors, got: %v", err)
	}
	if resp == nil || resp.Status != enums.DecisionRuleStatusActive {
		t.Fatalf("expected ACTIVE response despite publish failure, got %+v", resp)
	}
}

// errFakePublish is a sentinel error used by the publish-failure test.
var errFakePublish = &fakeErr{msg: "publish blew up"}

type fakeErr struct{ msg string }

func (e *fakeErr) Error() string { return e.msg }
