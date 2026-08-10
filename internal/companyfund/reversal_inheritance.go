package companyfund

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const applyReversalClassificationInheritanceSQL = `
WITH system_write AS (
	SELECT set_config('monera.company_fund_classification_origin', 'SYSTEM', true)
)
UPDATE company_fund_transactions reversal
SET finance_category_level1_id = original.finance_category_level1_id,
	finance_category_level2_id = original.finance_category_level2_id,
	is_operating_income_expense = original.is_operating_income_expense,
	summary_inclusion_override = original.summary_inclusion_override,
	classification_status = original.classification_status,
	classification_source = 'INHERITED_REVERSAL',
	classification_policy_version = $2,
	classification_updated_by = NULL,
	classification_updated_at = clock_timestamp(),
	updated_at = clock_timestamp()
FROM company_fund_transactions original, system_write
WHERE reversal.id = $1
  AND reversal.channel = 'AIRWALLEX'
  AND reversal.movement_kind = 'REVERSAL'
  AND reversal.reversal_of_transaction_id = original.id
  AND reversal.classification_source <> 'MANUAL'
  AND (
	reversal.finance_category_level1_id IS DISTINCT FROM original.finance_category_level1_id
	OR reversal.finance_category_level2_id IS DISTINCT FROM original.finance_category_level2_id
	OR reversal.is_operating_income_expense IS DISTINCT FROM original.is_operating_income_expense
	OR reversal.summary_inclusion_override IS DISTINCT FROM original.summary_inclusion_override
	OR reversal.classification_status IS DISTINCT FROM original.classification_status
	OR reversal.classification_source <> 'INHERITED_REVERSAL'
	OR reversal.classification_policy_version IS DISTINCT FROM $2
  )
RETURNING reversal.id`

const synchronizeReversalClassificationInheritanceSQL = `
WITH system_write AS (
	SELECT set_config('monera.company_fund_classification_origin', 'SYSTEM', true)
), candidates AS (
	SELECT reversal.id
	FROM company_fund_transactions reversal
	JOIN company_fund_transactions original
	  ON original.id = reversal.reversal_of_transaction_id
	WHERE reversal.channel = 'AIRWALLEX'
	  AND reversal.movement_kind = 'REVERSAL'
	  AND reversal.classification_source <> 'MANUAL'
	  AND (
		reversal.finance_category_level1_id IS DISTINCT FROM original.finance_category_level1_id
		OR reversal.finance_category_level2_id IS DISTINCT FROM original.finance_category_level2_id
		OR reversal.is_operating_income_expense IS DISTINCT FROM original.is_operating_income_expense
		OR reversal.summary_inclusion_override IS DISTINCT FROM original.summary_inclusion_override
		OR reversal.classification_status IS DISTINCT FROM original.classification_status
		OR reversal.classification_source <> 'INHERITED_REVERSAL'
		OR reversal.classification_policy_version IS DISTINCT FROM $1
	  )
	ORDER BY reversal.id
	LIMIT $2
	FOR UPDATE OF reversal SKIP LOCKED
)
UPDATE company_fund_transactions reversal
SET finance_category_level1_id = original.finance_category_level1_id,
	finance_category_level2_id = original.finance_category_level2_id,
	is_operating_income_expense = original.is_operating_income_expense,
	summary_inclusion_override = original.summary_inclusion_override,
	classification_status = original.classification_status,
	classification_source = 'INHERITED_REVERSAL',
	classification_policy_version = $1,
	classification_updated_by = NULL,
	classification_updated_at = clock_timestamp(),
	updated_at = clock_timestamp()
FROM candidates, company_fund_transactions original, system_write
WHERE reversal.id = candidates.id
  AND original.id = reversal.reversal_of_transaction_id
RETURNING reversal.id`

func (r *DBRepository) ApplyReversalClassificationInheritance(
	ctx context.Context,
	reversalTransactionID int64,
	policyVersion string,
) (bool, error) {
	if err := r.requireDB(); err != nil {
		return false, err
	}
	policyVersion = strings.TrimSpace(policyVersion)
	if reversalTransactionID <= 0 || policyVersion == "" || len(policyVersion) > 64 {
		return false, fmt.Errorf("invalid reversal classification inheritance request")
	}
	var updated int64
	if err := r.db.QueryRowContext(ctx, applyReversalClassificationInheritanceSQL,
		reversalTransactionID, policyVersion,
	).Scan(&updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("apply reversal classification inheritance: %w", err)
	}
	return true, nil
}

func (r *DBRepository) SynchronizeReversalClassificationInheritances(
	ctx context.Context,
	policyVersion string,
	limit int,
) (int, error) {
	if err := r.requireDB(); err != nil {
		return 0, err
	}
	policyVersion = strings.TrimSpace(policyVersion)
	if policyVersion == "" || len(policyVersion) > 64 || limit <= 0 || limit > 1000 {
		return 0, fmt.Errorf("invalid reversal inheritance maintenance bounds")
	}
	rows, err := r.db.QueryContext(ctx, synchronizeReversalClassificationInheritanceSQL, policyVersion, limit)
	if err != nil {
		return 0, fmt.Errorf("synchronize reversal classification inheritance: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return count, fmt.Errorf("scan synchronized reversal: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("iterate synchronized reversals: %w", err)
	}
	return count, nil
}
