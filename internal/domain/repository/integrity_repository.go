package repository

import (
	"context"

	"github.com/google/uuid"
)

// IntegrityRepository provides read/write operations for the attribute integrity checker.
type IntegrityRepository interface {
	// FindDecisionRulesWithInactiveAttributes returns IDs of non-INACTIVE
	// DecisionRules that reference at least one deactivated attribute via
	// their RuleConditions.
	FindDecisionRulesWithInactiveAttributes(ctx context.Context) ([]uuid.UUID, error)

	// FindDecisionRulesWithInvalidValues returns IDs of non-INACTIVE
	// DecisionRules where a RuleAttribute value is no longer present in
	// its Attribute's allowed values JSON.
	FindDecisionRulesWithInvalidValues(ctx context.Context) ([]uuid.UUID, error)

	// MarkDecisionRulesInactive bulk-sets STATUS = INACTIVE and
	// SUB_STATUS = "Missing attribute registry" for the given IDs.
	MarkDecisionRulesInactive(ctx context.Context, ids []uuid.UUID) error

	// TryAcquireCheckerLock attempts to acquire a Postgres advisory lock so
	// only one checker instance runs at a time. Returns false when already held.
	TryAcquireCheckerLock(ctx context.Context) (bool, error)

	// ReleaseCheckerLock releases the Postgres advisory lock.
	ReleaseCheckerLock(ctx context.Context) error
}
