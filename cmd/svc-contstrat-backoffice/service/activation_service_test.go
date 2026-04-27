package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/pubsub"
)

// recordingCacheRepo captures Publish payloads.
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

// errFakePublish is a sentinel error used by the publish-failure test.
var errFakePublish = &fakeErr{msg: "publish blew up"}

type fakeErr struct{ msg string }

func (e *fakeErr) Error() string { return e.msg }

// ---------------------------------------------------------------------------
// Tests for ActivationService.ActivatePublish
// ---------------------------------------------------------------------------

func TestActivatePublish_PublishesOnePingPerPlacement(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()
	cache := &recordingCacheRepo{}
	publisher := pubsub.NewPublisher(cache)
	svc := NewActivationService(publisher)

	schedules := []*entity.Schedule{
		scheduleForPlacement("home_hero"),
		scheduleForPlacement("checkout_banner"),
		scheduleForPlacement("home_hero"), // duplicate — should be deduped
	}

	svc.ActivatePublish(context.Background(), ruleID, schedules)

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
		if err := json.Unmarshal([]byte(payloads[i]), &msg); err != nil {
			t.Fatalf("publish[%d] payload not valid JSON: %v", i, err)
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

func TestActivatePublish_NilPublisherIsNoOp(t *testing.T) {
	t.Parallel()

	svc := NewActivationService(nil)
	// Must not panic.
	svc.ActivatePublish(context.Background(), uuid.New(), []*entity.Schedule{
		scheduleForPlacement("home_hero"),
	})
}

func TestActivatePublish_EmptySchedulesIsNoOp(t *testing.T) {
	t.Parallel()

	cache := &recordingCacheRepo{}
	svc := NewActivationService(pubsub.NewPublisher(cache))

	svc.ActivatePublish(context.Background(), uuid.New(), nil)
	svc.ActivatePublish(context.Background(), uuid.New(), []*entity.Schedule{})

	channels, _ := cache.snapshot()
	if len(channels) != 0 {
		t.Fatalf("expected no publishes for empty schedules, got %d", len(channels))
	}
}

func TestActivatePublish_SkipsNilAndBlankPlacementSchedules(t *testing.T) {
	t.Parallel()

	cache := &recordingCacheRepo{}
	svc := NewActivationService(pubsub.NewPublisher(cache))

	nilEntry := (*entity.Schedule)(nil)
	noPlacement := &entity.Schedule{BaseModel: entity.BaseModel{ID: uuid.New()}}
	blankName := &entity.Schedule{
		BaseModel: entity.BaseModel{ID: uuid.New()},
		Placement: &entity.Placement{PlacementName: ""},
	}
	valid := scheduleForPlacement("sidebar_ad")

	svc.ActivatePublish(context.Background(), uuid.New(), []*entity.Schedule{
		nilEntry, noPlacement, blankName, valid,
	})

	channels, _ := cache.snapshot()
	if len(channels) != 1 {
		t.Fatalf("expected exactly 1 publish (valid placement only), got %d", len(channels))
	}
}

func TestActivatePublish_PublishErrorDoesNotPanic(t *testing.T) {
	t.Parallel()

	cache := &recordingCacheRepo{publishErr: errFakePublish}
	svc := NewActivationService(pubsub.NewPublisher(cache))

	// Must not panic or return any error (errors are swallowed by design).
	svc.ActivatePublish(context.Background(), uuid.New(), []*entity.Schedule{
		scheduleForPlacement("home_hero"),
	})

	channels, _ := cache.snapshot()
	if len(channels) != 1 {
		t.Fatalf("expected Publish to be attempted once even on error, got %d", len(channels))
	}
}
