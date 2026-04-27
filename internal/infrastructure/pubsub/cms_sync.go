// Package pubsub holds the shared Redis Pub/Sub contract between the
// backoffice (publisher) and delivery (subscriber) services.
//
// The CMS sync channel carries SyncPingMessage payloads that signal a
// cache-invalidating change in the backoffice (e.g. a decision-rule
// activation). Delivery pods subscribe to the channel and refresh the
// affected placement on receipt.
package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/logger"
)

// ChannelCMSSyncPing is the Redis Pub/Sub channel name used for cache
// invalidation pings between backoffice and delivery.
const ChannelCMSSyncPing = "cms:sync:ping"

// SyncPingMessage is the structured payload published on ChannelCMSSyncPing.
//
//   - PlacementName scopes the invalidation to a single placement; when empty
//     the subscriber falls back to a full evaluate.
//   - VersionHash lets subscribers skip work when their local version already
//     matches the publisher's; an empty hash forces a refresh.
//   - DecisionRuleID, when set, instructs the subscriber to delete the
//     `rule:{id}` cache entry explicitly before re-evaluating. Optional and
//     backwards compatible — older publishers omit the field.
type SyncPingMessage struct {
	PlacementName  string `json:"placement_name"`
	VersionHash    string `json:"version_hash"`
	DecisionRuleID string `json:"decision_rule_id,omitempty"`
}

// Publisher publishes SyncPingMessage payloads to the CMS sync channel.
//
// A nil receiver or a Publisher constructed with a nil RedisCacheRepository
// is a no-op — callers can safely invoke PingPlacement without conditionals
// even in environments where Redis is unavailable.
type Publisher struct {
	redis domainrepo.RedisCacheRepository
}

// NewPublisher returns a Publisher that writes to the provided Redis
// repository. Passing nil yields a no-op publisher.
func NewPublisher(redis domainrepo.RedisCacheRepository) *Publisher {
	return &Publisher{redis: redis}
}

// PingPlacement publishes a SyncPingMessage to ChannelCMSSyncPing. Pass an
// empty versionHash to force subscribers to refresh; pass a non-empty
// decisionRuleID to have subscribers explicitly delete the cached rule.
//
// Errors are returned to the caller but are also logged so fire-and-forget
// usage at HTTP request boundaries can ignore them safely.
func (p *Publisher) PingPlacement(ctx context.Context, placementName, decisionRuleID, versionHash string) error {
	if p == nil || p.redis == nil {
		return nil
	}

	payload, err := json.Marshal(SyncPingMessage{
		PlacementName:  placementName,
		VersionHash:    versionHash,
		DecisionRuleID: decisionRuleID,
	})
	if err != nil {
		return fmt.Errorf("marshal sync ping: %w", err)
	}

	if err := p.redis.Publish(ctx, ChannelCMSSyncPing, string(payload)); err != nil {
		logger.LSystem(ctx, entity.SystemLog{
			Service: "CMS-PUBSUB",
			Level:   "WARN",
			Message: fmt.Sprintf("publisher: ping for placement=%q rule=%q failed: %v", placementName, decisionRuleID, err),
		})
		return fmt.Errorf("publish sync ping: %w", err)
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "CMS-PUBSUB",
		Level:   "INFO",
		Message: fmt.Sprintf("publisher: ping sent for placement=%q rule=%q", placementName, decisionRuleID),
	})
	return nil
}
