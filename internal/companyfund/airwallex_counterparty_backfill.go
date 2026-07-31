package companyfund

import (
	"context"
	"fmt"
)

const airwallexPayoutCounterpartyBackfillCandidateSQL = `
FROM company_fund_transactions transaction
	JOIN company_fund_provider_transaction_facts fact
	  ON fact.id = transaction.provider_transaction_fact_id
	JOIN company_fund_provider_events event
	  ON event.id = fact.source_provider_event_id
	WHERE transaction.channel = 'AIRWALLEX'
	  AND transaction.movement_kind = 'PRINCIPAL'
	  AND transaction.transaction_direction = 'OUTFLOW'
	  AND transaction.to_address_or_account IS NULL
	  AND fact.channel = 'AIRWALLEX'
	  AND upper(COALESCE(fact.provider_extras->>'transaction_type', '')) = 'PAYOUT'
	  AND upper(COALESCE(fact.provider_extras->>'source_type', '')) = 'PAYOUT'
	  AND event.channel = 'AIRWALLEX'
	  AND event.event_type = 'FINANCIAL_TRANSACTION_SNAPSHOT'
	  AND event.event_state = 'PROCESSED'
	  AND event.source_kind = 'OWNED_ENCRYPTED_PAYLOAD'
	  AND event.owned_payload_purged_at IS NULL
	  AND (
	    event.owned_payload_legal_hold = true
	    OR event.owned_payload_retention_until > NOW()
	  )`

const requeueAirwallexPayoutCounterpartyBackfillSQL = `
WITH candidates AS (
	SELECT event.id
	` + airwallexPayoutCounterpartyBackfillCandidateSQL + `
	ORDER BY event.id
	LIMIT $1
	FOR UPDATE OF event SKIP LOCKED
)
UPDATE company_fund_provider_events event
SET event_state = 'PENDING',
	processed_at = NULL,
	next_attempt_at = NULL,
	last_error = NULL,
	updated_at = NOW()
FROM candidates
WHERE event.id = candidates.id
  AND event.event_state = 'PROCESSED'
RETURNING event.id`

const hasAirwallexPayoutCounterpartyBackfillCandidatesSQL = `
SELECT EXISTS (
	SELECT 1
	` + airwallexPayoutCounterpartyBackfillCandidateSQL + `
)`

// RequeueAirwallexPayoutCounterpartyBackfill reuses retained, already
// processed Financial Transactions snapshots to enrich historical PAYOUT
// movements through the same idempotent provider-event path as new data.
func (r *DBRepository) RequeueAirwallexPayoutCounterpartyBackfill(
	ctx context.Context,
	limit int,
) (int, error) {
	if err := r.requireDB(); err != nil {
		return 0, err
	}
	if limit < 1 || limit > 1000 {
		return 0, fmt.Errorf("invalid Airwallex payout counterparty backfill limit")
	}
	rows, err := r.db.QueryContext(ctx, requeueAirwallexPayoutCounterpartyBackfillSQL, limit)
	if err != nil {
		return 0, fmt.Errorf("requeue Airwallex payout counterparty backfill: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var eventID int64
		if err := rows.Scan(&eventID); err != nil {
			return 0, fmt.Errorf("scan Airwallex payout counterparty backfill event: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate Airwallex payout counterparty backfill events: %w", err)
	}
	return count, nil
}

// HasAirwallexPayoutCounterpartyBackfillCandidates distinguishes a truly
// drained backfill from an empty SKIP LOCKED batch while another instance owns
// a candidate row.
func (r *DBRepository) HasAirwallexPayoutCounterpartyBackfillCandidates(
	ctx context.Context,
) (bool, error) {
	if err := r.requireDB(); err != nil {
		return false, err
	}
	var found bool
	if err := r.db.QueryRowContext(
		ctx,
		hasAirwallexPayoutCounterpartyBackfillCandidatesSQL,
	).Scan(&found); err != nil {
		return false, fmt.Errorf("check Airwallex payout counterparty backfill candidates: %w", err)
	}
	return found, nil
}
