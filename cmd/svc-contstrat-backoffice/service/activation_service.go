package service

import (
	"context"
	"fmt"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/infrastructure/pubsub"

	"github.com/google/uuid"
)

type ActivationService struct {
	publisher *pubsub.Publisher
}

func NewActivationService(publisher *pubsub.Publisher) *ActivationService {
	return &ActivationService{publisher: publisher}
}

// ActivatePublish publishes a sync ping per distinct placement
// touched by the activated decision rule so delivery pods drop their cached
// `schedules:placement:{name}` and `rule:{id}` entries and re-evaluate.
//
// Failures are logged and swallowed: the activation has already succeeded,
// and stale entries will fall out of cache via TTL.
func (s *ActivationService) ActivatePublish(ctx context.Context, ruleID uuid.UUID, schedules []*entity.Schedule) {
	if s.publisher == nil || len(schedules) == 0 {
		return
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "DECISION-RULE-WIZARD",
		Level:   "INFO",
		Message: fmt.Sprintf("activate: publishing cache-invalidation pings for rule=%s placements=%d", ruleID, len(schedules)),
	})

	seen := make(map[string]struct{}, len(schedules))
	for _, sc := range schedules {
		if sc == nil || sc.Placement == nil {
			continue
		}
		name := sc.Placement.PlacementName
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}

		// Empty version hash → forces subscriber to refresh regardless of
		// any previously-mirrored version.
		if err := s.publisher.PingSync(ctx, name, ruleID.String(), ""); err != nil {
			logger.LSystem(ctx, entity.SystemLog{
				Service: "DECISION-RULE-WIZARD",
				Level:   "WARN",
				Message: fmt.Sprintf("activate: cache-invalidation ping failed for placement=%q rule=%s: %v", name, ruleID, err),
			})
		}
	}
}
