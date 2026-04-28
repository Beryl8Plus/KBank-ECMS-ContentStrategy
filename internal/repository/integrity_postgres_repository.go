package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// checkerAdvisoryLockID is the Postgres session-level advisory lock key used to
// ensure only one integrity checker runs at a time across all instances.
const checkerAdvisoryLockID = 8823746291 // arbitrary stable int64

// IntegrityPostgresRepository implements domainrepo.IntegrityRepository.
type IntegrityPostgresRepository struct {
	db *gorm.DB
}

// Compile-time interface check.
var _ domainrepo.IntegrityRepository = (*IntegrityPostgresRepository)(nil)

// NewIntegrityPostgresRepository creates a new IntegrityPostgresRepository.
func NewIntegrityPostgresRepository(db *gorm.DB) *IntegrityPostgresRepository {
	return &IntegrityPostgresRepository{db: db}
}

// FindDecisionRulesWithInactiveAttributes returns IDs of non-INACTIVE DecisionRules
// that reference a deactivated attribute through their RuleConditions.
func (r *IntegrityPostgresRepository) FindDecisionRulesWithInactiveAttributes(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT rc."DECISION_RULE_ID"
		FROM rule_conditions rc
		JOIN attributes a
		  ON a."ID" = rc."ATTRIBUTE_ID"
		 AND a."DELETED_AT" IS NULL
		 AND a."IS_ACTIVE" = false
		JOIN decision_rules dr
		  ON dr."ID" = rc."DECISION_RULE_ID"
		 AND dr."DELETED_AT" IS NULL
		 AND dr."STATUS" != ?
		WHERE rc."DELETED_AT" IS NULL
	`, enums.DecisionRuleStatusInactive).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("find DRs with inactive attributes: %w", err)
	}
	return ids, nil
}

// FindDecisionRulesWithInvalidValues returns IDs of non-INACTIVE DecisionRules
// where a RuleAttribute's stored value is no longer present in its Attribute's
// allowed values JSON array.
//
// Uses Postgres jsonb containment (@>) — works for both ["val1","val2"] and
// [{"value":"val1"},{"value":"val2"}] formats as long as RuleAttribute.Value
// is stored in the matching element format.
func (r *IntegrityPostgresRepository) FindDecisionRulesWithInvalidValues(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT r."DECISION_RULE_ID"
		FROM rule_attributes ra
		JOIN attributes a
		  ON a."ID" = ra."ATTRIBUTE_ID"
		 AND a."DELETED_AT" IS NULL
		JOIN rules r
		  ON r."ID" = ra."RULE_ID"
		 AND r."DELETED_AT" IS NULL
		JOIN decision_rules dr
		  ON dr."ID" = r."DECISION_RULE_ID"
		 AND dr."DELETED_AT" IS NULL
		 AND dr."STATUS" != ?
		WHERE ra."DELETED_AT" IS NULL
		  AND a."VALUE" IS NOT NULL
		  AND jsonb_typeof(a."VALUE") = 'array'
		  AND jsonb_array_length(a."VALUE") > 0
		  AND ra."VALUE" IS NOT NULL
		  AND NOT (a."VALUE" @> ra."VALUE")
	`, enums.DecisionRuleStatusInactive).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("find DRs with invalid values: %w", err)
	}
	return ids, nil
}

// MarkDecisionRulesInactive bulk-sets STATUS = INACTIVE and
// SUB_STATUS = "Missing attribute registry" for the given DecisionRule IDs.
func (r *IntegrityPostgresRepository) MarkDecisionRulesInactive(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).
		Model(&entity.DecisionRule{}).
		Where(`"ID" IN ? AND "DELETED_AT" IS NULL`, ids).
		Updates(map[string]any{
			"STATUS":     enums.DecisionRuleStatusInactive,
			"SUB_STATUS": enums.DecisionRuleSubStatusMissing,
		}).Error
	if err != nil {
		return fmt.Errorf("mark decision rules inactive: %w", err)
	}
	return nil
}

// TryAcquireCheckerLock attempts to acquire a Postgres session-level advisory lock.
// Returns false when the lock is already held by another session (non-blocking).
func (r *IntegrityPostgresRepository) TryAcquireCheckerLock(ctx context.Context) (bool, error) {
	var acquired bool
	if err := r.db.WithContext(ctx).
		Raw("SELECT pg_try_advisory_lock(?)", checkerAdvisoryLockID).
		Scan(&acquired).Error; err != nil {
		return false, fmt.Errorf("try acquire checker lock: %w", err)
	}
	return acquired, nil
}

// ReleaseCheckerLock releases the Postgres session-level advisory lock.
func (r *IntegrityPostgresRepository) ReleaseCheckerLock(ctx context.Context) error {
	if err := r.db.WithContext(ctx).
		Exec("SELECT pg_advisory_unlock(?)", checkerAdvisoryLockID).Error; err != nil {
		return fmt.Errorf("release checker lock: %w", err)
	}
	return nil
}
