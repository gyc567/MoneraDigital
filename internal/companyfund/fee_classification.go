package companyfund

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type AirwallexFeeClassificationPolicy struct {
	Level1Code    string
	Level2Code    string
	PolicyVersion string
}

type financeCategoryPair struct {
	Level1ID int64
	Level2ID int64
}

var errAirwallexFeeCategoryUnavailable = errors.New(
	"configured Airwallex fee category hierarchy is unavailable",
)

const resolveAirwallexFeeCategoriesSQL = `
SELECT level1.id, level2.id
FROM finance_categories level1
JOIN finance_categories level2 ON level2.parent_id = level1.id
WHERE level1.code = $1
  AND level1.level = 1
  AND level1.is_enabled
  AND level2.code = $2
  AND level2.level = 2
  AND level2.is_enabled`

const publishAirwallexFeeClassificationBindingSQL = `
INSERT INTO company_fund_classification_policy_bindings (
	policy_key,
	channel,
	movement_kind,
	finance_category_level1_id,
	finance_category_level2_id,
	policy_version,
	is_active,
	updated_at
) VALUES (
	'AIRWALLEX_FEE',
	'AIRWALLEX',
	'FEE',
	$1,
	$2,
	$3,
	TRUE,
	clock_timestamp()
)
ON CONFLICT (policy_key) DO UPDATE
SET finance_category_level1_id = EXCLUDED.finance_category_level1_id,
	finance_category_level2_id = EXCLUDED.finance_category_level2_id,
	policy_version = EXCLUDED.policy_version,
	is_active = TRUE,
	updated_at = clock_timestamp()`

const applyAirwallexFeeClassificationSQL = `
WITH system_write AS (
	SELECT set_config('monera.company_fund_classification_origin', 'SYSTEM', true)
)
UPDATE company_fund_transactions
SET finance_category_level1_id = $2,
	finance_category_level2_id = $3,
	is_operating_income_expense = TRUE,
	summary_inclusion_override = TRUE,
	classification_status = 'CLASSIFIED',
	classification_source = 'AUTO_RULE',
	classification_policy_version = $4,
	classification_updated_by = NULL,
	classification_updated_at = clock_timestamp(),
	updated_at = clock_timestamp()
FROM system_write
WHERE id = $1
  AND channel = 'AIRWALLEX'
  AND movement_kind = 'FEE'
  AND classification_source IN ('UNCLASSIFIED', 'AUTO_RULE')
RETURNING id`

const listAirwallexFeesNeedingClassificationSQL = `
SELECT transaction.id,
	transaction.provider_transaction_fact_id,
	transaction.provider_account_key,
	transaction.provider_transaction_id
FROM company_fund_transactions transaction
WHERE transaction.channel = 'AIRWALLEX'
  AND transaction.movement_kind = 'FEE'
  AND transaction.provider_transaction_fact_id IS NOT NULL
  AND transaction.provider_account_key IS NOT NULL
  AND transaction.provider_transaction_id IS NOT NULL
  AND (
	transaction.classification_source = 'UNCLASSIFIED'
	OR (
	  transaction.classification_source = 'AUTO_RULE'
	  AND transaction.classification_policy_version IS DISTINCT FROM $1
	)
  )
  AND NOT EXISTS (
	SELECT 1
	FROM company_fund_ledger_tasks task
	WHERE task.channel = 'AIRWALLEX'
	  AND task.task_kind = 'FEE_CLASSIFICATION'
	  AND task.subject_transaction_id = transaction.id
	  AND COALESCE(task.policy_version, '') = COALESCE($1, '')
  )
ORDER BY transaction.id
LIMIT $2`

func (policy AirwallexFeeClassificationPolicy) validate() error {
	for label, value := range map[string]string{
		"Airwallex fee level-one category code":       policy.Level1Code,
		"Airwallex fee level-two category code":       policy.Level2Code,
		"Airwallex fee classification policy version": policy.PolicyVersion,
	} {
		if err := validateRequiredString(label, strings.TrimSpace(value), 128); err != nil {
			return err
		}
	}
	return nil
}

func (r *DBRepository) resolveAirwallexFeeCategories(
	ctx context.Context,
	policy AirwallexFeeClassificationPolicy,
) (financeCategoryPair, error) {
	if err := policy.validate(); err != nil {
		return financeCategoryPair{}, err
	}
	var pair financeCategoryPair
	if err := r.db.QueryRowContext(ctx, resolveAirwallexFeeCategoriesSQL,
		strings.TrimSpace(policy.Level1Code), strings.TrimSpace(policy.Level2Code),
	).Scan(&pair.Level1ID, &pair.Level2ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return financeCategoryPair{}, errAirwallexFeeCategoryUnavailable
		}
		return financeCategoryPair{}, fmt.Errorf("resolve Airwallex fee category hierarchy: %w", err)
	}
	return pair, nil
}

func (r *DBRepository) publishAirwallexFeeClassificationBinding(
	ctx context.Context,
	pair financeCategoryPair,
	policyVersion string,
) error {
	if _, err := r.db.ExecContext(
		ctx,
		publishAirwallexFeeClassificationBindingSQL,
		pair.Level1ID,
		pair.Level2ID,
		strings.TrimSpace(policyVersion),
	); err != nil {
		return fmt.Errorf("publish Airwallex fee classification binding: %w", err)
	}
	return nil
}

func (r *DBRepository) ApplyAirwallexFeeClassification(
	ctx context.Context,
	transactionID int64,
	policy AirwallexFeeClassificationPolicy,
) (bool, error) {
	if err := r.requireDB(); err != nil {
		return false, err
	}
	if transactionID <= 0 {
		return false, fmt.Errorf("Airwallex fee transaction ID must be positive")
	}
	pair, err := r.resolveAirwallexFeeCategories(ctx, policy)
	if err != nil {
		return false, err
	}
	if err := r.publishAirwallexFeeClassificationBinding(ctx, pair, policy.PolicyVersion); err != nil {
		return false, err
	}
	var updated int64
	if err := r.db.QueryRowContext(ctx, applyAirwallexFeeClassificationSQL,
		transactionID, pair.Level1ID, pair.Level2ID, strings.TrimSpace(policy.PolicyVersion),
	).Scan(&updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("apply Airwallex fee classification: %w", err)
	}
	return true, nil
}

// EnqueueAirwallexFeeClassificationBackfill is bounded and restart-safe. The
// task identity prevents repeated scans from producing duplicate work.
func (r *DBRepository) EnqueueAirwallexFeeClassificationBackfill(
	ctx context.Context,
	policy AirwallexFeeClassificationPolicy,
	limit int,
	taskSLA time.Duration,
) (int, error) {
	if err := r.requireDB(); err != nil {
		return 0, err
	}
	if err := validateRequiredString(
		"Airwallex fee classification backfill policy version",
		strings.TrimSpace(policy.PolicyVersion),
		64,
	); err != nil {
		return 0, err
	}
	if limit <= 0 || limit > 1000 || taskSLA <= 0 {
		return 0, fmt.Errorf("invalid Airwallex fee classification backfill bounds")
	}
	if strings.TrimSpace(policy.Level1Code) != "" && strings.TrimSpace(policy.Level2Code) != "" {
		pair, err := r.resolveAirwallexFeeCategories(ctx, policy)
		if err != nil && !errors.Is(err, errAirwallexFeeCategoryUnavailable) {
			return 0, err
		}
		if err == nil {
			if err := r.publishAirwallexFeeClassificationBinding(ctx, pair, policy.PolicyVersion); err != nil {
				return 0, err
			}
		}
	}

	rows, err := r.db.QueryContext(ctx, listAirwallexFeesNeedingClassificationSQL, policy.PolicyVersion, limit)
	if err != nil {
		return 0, fmt.Errorf("list Airwallex fees needing classification: %w", err)
	}
	type candidate struct {
		transactionID         int64
		factID                int64
		providerAccountKey    string
		providerTransactionID string
	}
	candidates := make([]candidate, 0, limit)
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.transactionID, &item.factID, &item.providerAccountKey, &item.providerTransactionID); err != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("scan Airwallex fee classification candidate: %w", err)
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, fmt.Errorf("iterate Airwallex fee classification candidates: %w", err)
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("close Airwallex fee classification candidates: %w", err)
	}
	enqueued := 0
	for _, item := range candidates {
		result, err := r.EnqueueCompanyFundLedgerTask(ctx, CompanyFundLedgerTaskInput{
			Channel:                      ChannelAirwallex,
			ProviderAccountKey:           item.providerAccountKey,
			Kind:                         LedgerTaskKindFeeClassification,
			ProviderTransactionFactID:    item.factID,
			SubjectProviderTransactionID: item.providerTransactionID,
			SubjectTransactionID:         &item.transactionID,
			EvidenceReference:            "finance-approved-airwallex-fee-default",
			PolicyVersion:                policy.PolicyVersion,
			RelationshipSLA:              taskSLA,
		})
		if err != nil {
			return enqueued, err
		}
		if result.Inserted {
			enqueued++
		}
	}
	return enqueued, nil
}
