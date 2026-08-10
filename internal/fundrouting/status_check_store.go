package fundrouting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type PostgresRoutingStatusCheckStore struct {
	db *sql.DB
}

func NewPostgresRoutingStatusCheckStore(db *sql.DB) (*PostgresRoutingStatusCheckStore, error) {
	if db == nil {
		return nil, fmt.Errorf("Safeheron routing status check database is required")
	}
	return &PostgresRoutingStatusCheckStore{db: db}, nil
}

func (s *PostgresRoutingStatusCheckStore) ScheduleOpen(ctx context.Context, initialDelay time.Duration) (int64, error) {
	if s == nil || s.db == nil || initialDelay <= 0 {
		return 0, fmt.Errorf("Safeheron routing status scheduling is not configured")
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_status_checks status_check
SET completed_at=now(),next_check_at=NULL,lease_owner=NULL,lease_expires_at=NULL,updated_at=now()
WHERE status_check.completed_at IS NULL
  AND (status_check.lease_owner IS NULL OR status_check.lease_expires_at<=now())
  AND NOT EXISTS (
    SELECT 1 FROM safeheron_transaction_routing_cases routing
    WHERE routing.safeheron_tx_key=status_check.safeheron_tx_key
      AND routing.decision='OPEN' AND routing.reason_code='STATUS_NOT_TERMINAL'
  )`); err != nil {
		return 0, fmt.Errorf("complete inactive Safeheron routing status checks: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_status_checks
  (safeheron_tx_key,first_seen_at,next_check_at)
SELECT routing.safeheron_tx_key,min(routing.created_at),
       min(routing.created_at)+($1 * interval '1 millisecond')
FROM safeheron_transaction_routing_cases routing
WHERE routing.decision='OPEN' AND routing.reason_code='STATUS_NOT_TERMINAL'
GROUP BY routing.safeheron_tx_key
ON CONFLICT (safeheron_tx_key) DO NOTHING`, initialDelay.Milliseconds())
	if err != nil {
		return 0, fmt.Errorf("schedule open Safeheron routing status checks: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("inspect scheduled Safeheron routing status checks: %w", err)
	}
	return rows, nil
}

func (s *PostgresRoutingStatusCheckStore) ClaimDue(
	ctx context.Context,
	workerID string,
	lease time.Duration,
) (RoutingStatusCheck, bool, error) {
	if s == nil || s.db == nil || strings.TrimSpace(workerID) == "" || lease <= 0 {
		return RoutingStatusCheck{}, false, fmt.Errorf("Safeheron routing status claim is not configured")
	}
	var check RoutingStatusCheck
	err := s.db.QueryRowContext(ctx, `WITH candidate AS (
  SELECT status_check.safeheron_tx_key
  FROM safeheron_transaction_routing_status_checks status_check
  WHERE status_check.completed_at IS NULL
    AND status_check.next_check_at<=now()
    AND (status_check.lease_owner IS NULL OR status_check.lease_expires_at<=now())
    AND EXISTS (
      SELECT 1 FROM safeheron_transaction_routing_cases routing
      WHERE routing.safeheron_tx_key=status_check.safeheron_tx_key
        AND routing.decision='OPEN' AND routing.reason_code='STATUS_NOT_TERMINAL'
    )
  ORDER BY status_check.next_check_at,status_check.safeheron_tx_key
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE safeheron_transaction_routing_status_checks status_check
SET lease_owner=$1,lease_expires_at=now()+($2 * interval '1 millisecond'),
    attempt_count=status_check.attempt_count+1,updated_at=now()
FROM candidate
WHERE status_check.safeheron_tx_key=candidate.safeheron_tx_key
RETURNING status_check.safeheron_tx_key,status_check.first_seen_at,
          status_check.attempt_count,status_check.lease_owner`, workerID, lease.Milliseconds()).Scan(
		&check.TxKey, &check.FirstSeenAt, &check.AttemptCount, &check.LeaseOwner,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingStatusCheck{}, false, nil
	}
	if err != nil {
		return RoutingStatusCheck{}, false, fmt.Errorf("claim due Safeheron routing status check: %w", err)
	}
	return check, true, nil
}

func (s *PostgresRoutingStatusCheckStore) CompleteObserved(ctx context.Context, observed routingStatusObserved) error {
	if s == nil || s.db == nil || strings.TrimSpace(observed.Check.TxKey) == "" ||
		strings.TrimSpace(observed.Check.LeaseOwner) == "" || observed.CheckedAt.IsZero() ||
		strings.TrimSpace(observed.Status) == "" || strings.TrimSpace(observed.EventID) == "" ||
		(observed.Terminal == (observed.NextCheckAt != nil)) {
		return fmt.Errorf("Safeheron routing status observation is invalid")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_status_checks
SET last_checked_at=$3::timestamptz,last_check_outcome='OBSERVED',last_observed_status=$4,
    last_provider_event_id=$7,last_error_code=NULL,
    next_check_at=CASE WHEN $5::boolean THEN NULL::timestamptz ELSE $6::timestamptz END,
    completed_at=CASE WHEN $5::boolean THEN $3::timestamptz ELSE NULL::timestamptz END,
    lease_owner=NULL,lease_expires_at=NULL,updated_at=now()
	WHERE safeheron_tx_key=$1 AND lease_owner=$2`, observed.Check.TxKey, observed.Check.LeaseOwner,
		observed.CheckedAt, observed.Status, observed.Terminal, observed.NextCheckAt, observed.EventID)
	if err != nil {
		return fmt.Errorf("persist Safeheron routing status observation: %w", err)
	}
	return requireRoutingStatusLease(result, observed.Check.TxKey)
}

func (s *PostgresRoutingStatusCheckStore) CompleteFailed(ctx context.Context, failure routingStatusFailure) error {
	if s == nil || s.db == nil || strings.TrimSpace(failure.Check.TxKey) == "" ||
		strings.TrimSpace(failure.Check.LeaseOwner) == "" || failure.CheckedAt.IsZero() ||
		strings.TrimSpace(failure.ErrorCode) == "" || failure.NextCheckAt.IsZero() {
		return fmt.Errorf("Safeheron routing status failure is invalid")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_status_checks
SET last_checked_at=$3,last_check_outcome='ERROR',last_provider_event_id=NULL,last_error_code=$4,next_check_at=$5,
    lease_owner=NULL,lease_expires_at=NULL,updated_at=now()
WHERE safeheron_tx_key=$1 AND lease_owner=$2`, failure.Check.TxKey, failure.Check.LeaseOwner,
		failure.CheckedAt, failure.ErrorCode, failure.NextCheckAt)
	if err != nil {
		return fmt.Errorf("persist Safeheron routing status failure: %w", err)
	}
	return requireRoutingStatusLease(result, failure.Check.TxKey)
}

func requireRoutingStatusLease(result sql.Result, txKey string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect Safeheron routing status lease: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("Safeheron routing status lease was lost for transaction %s", txKey)
	}
	return nil
}

func (s *PostgresRoutingStatusCheckStore) NextDue(ctx context.Context, initialDelay time.Duration) (time.Time, error) {
	if s == nil || s.db == nil || initialDelay <= 0 {
		return time.Time{}, fmt.Errorf("Safeheron routing status due calculation is not configured")
	}
	var due sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT min(due_at) FROM (
  SELECT CASE
    WHEN status_check.lease_owner IS NOT NULL AND status_check.lease_expires_at>now()
      THEN status_check.lease_expires_at
    ELSE status_check.next_check_at
  END AS due_at
  FROM safeheron_transaction_routing_status_checks status_check
  WHERE status_check.completed_at IS NULL
    AND EXISTS (
      SELECT 1 FROM safeheron_transaction_routing_cases routing
      WHERE routing.safeheron_tx_key=status_check.safeheron_tx_key
        AND routing.decision='OPEN' AND routing.reason_code='STATUS_NOT_TERMINAL'
    )
  UNION ALL
  SELECT min(routing.created_at)+($1 * interval '1 millisecond') AS due_at
  FROM safeheron_transaction_routing_cases routing
  WHERE routing.decision='OPEN' AND routing.reason_code='STATUS_NOT_TERMINAL'
    AND NOT EXISTS (
      SELECT 1 FROM safeheron_transaction_routing_status_checks status_check
      WHERE status_check.safeheron_tx_key=routing.safeheron_tx_key
    )
  GROUP BY routing.safeheron_tx_key
) deadlines`, initialDelay.Milliseconds()).Scan(&due)
	if err != nil || !due.Valid {
		return time.Time{}, err
	}
	return due.Time, nil
}

var _ RoutingStatusCheckStore = (*PostgresRoutingStatusCheckStore)(nil)
