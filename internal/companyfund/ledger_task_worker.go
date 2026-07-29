package companyfund

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const claimCompanyFundLedgerTaskSQL = `
WITH candidate AS (
	SELECT id
	FROM company_fund_ledger_tasks
	WHERE (
		task_state = 'WAITING'
		AND (
			next_attempt_at <= clock_timestamp()
			OR sla_expires_at <= clock_timestamp()
		)
	) OR (
		task_state = 'LEASED' AND lease_expires_at <= clock_timestamp()
	)
	ORDER BY next_attempt_at, id
	FOR UPDATE SKIP LOCKED
	LIMIT 1
)
UPDATE company_fund_ledger_tasks task
SET task_state = 'LEASED',
	lease_owner = $1,
	lease_expires_at = clock_timestamp() + ($2::bigint * INTERVAL '1 microsecond'),
	attempt_count = attempt_count + 1,
	updated_at = clock_timestamp()
FROM candidate
WHERE task.id = candidate.id
RETURNING task.id,
	task.channel,
	task.provider_account_key,
	task.task_kind,
	task.provider_transaction_fact_id,
	task.subject_provider_transaction_id,
	task.subject_transaction_id,
	COALESCE(task.relationship_reference_type, ''),
	COALESCE(task.relationship_reference_key, ''),
	COALESCE(task.relationship_group_key, ''),
	task.evidence_reference,
	task.task_payload,
	task.task_payload_digest,
	COALESCE(task.policy_version, ''),
	task.attempt_count,
	task.sla_expires_at`

const completeCompanyFundLedgerTaskSQL = `
UPDATE company_fund_ledger_tasks
SET task_state = 'COMPLETED',
	subject_transaction_id = COALESCE($3, subject_transaction_id),
	lease_owner = NULL,
	lease_expires_at = NULL,
	terminal_at = clock_timestamp(),
	last_error_code = NULL,
	updated_at = clock_timestamp()
WHERE id = $1
  AND task_state = 'LEASED'
  AND lease_owner = $2
  AND attempt_count = $4
  AND lease_expires_at > clock_timestamp()
RETURNING id`

const retryCompanyFundLedgerTaskSQL = `
UPDATE company_fund_ledger_tasks
SET task_state = CASE WHEN sla_expires_at <= clock_timestamp() THEN 'DEAD_LETTER' ELSE 'WAITING' END,
	lease_owner = NULL,
	lease_expires_at = NULL,
	next_attempt_at = CASE
		WHEN sla_expires_at <= clock_timestamp() THEN next_attempt_at
		ELSE clock_timestamp() + ($3::bigint * INTERVAL '1 microsecond')
	END,
	terminal_at = CASE WHEN sla_expires_at <= clock_timestamp() THEN clock_timestamp() ELSE NULL END,
	last_error_code = $4,
	updated_at = clock_timestamp()
WHERE id = $1
  AND task_state = 'LEASED'
  AND lease_owner = $2
  AND attempt_count = $5
  AND lease_expires_at > clock_timestamp()
RETURNING task_state`

const deadLetterCompanyFundLedgerTaskSQL = `
UPDATE company_fund_ledger_tasks
SET task_state = 'DEAD_LETTER',
	lease_owner = NULL,
	lease_expires_at = NULL,
	terminal_at = clock_timestamp(),
	last_error_code = $3,
	updated_at = clock_timestamp()
WHERE id = $1
  AND task_state = 'LEASED'
  AND lease_owner = $2
  AND attempt_count = $4
  AND lease_expires_at > clock_timestamp()
RETURNING id`

const nextCompanyFundLedgerTaskDueSQL = `
SELECT MIN(
	CASE
		WHEN task_state = 'WAITING' THEN LEAST(next_attempt_at, sla_expires_at)
		ELSE lease_expires_at
	END
)
FROM company_fund_ledger_tasks
WHERE task_state IN ('WAITING', 'LEASED')`

type CompanyFundLedgerTaskLease struct {
	ID                           int64
	Channel                      TransactionSource
	ProviderAccountKey           string
	Kind                         LedgerTaskKind
	ProviderTransactionFactID    int64
	SubjectProviderTransactionID string
	SubjectTransactionID         *int64
	RelationshipReferenceType    RelationshipReferenceType
	RelationshipReferenceKey     string
	RelationshipGroupKey         string
	EvidenceReference            string
	Proposal                     TransactionUpsertInput
	PolicyVersion                string
	AttemptCount                 int
	SLAExpiresAt                 time.Time
}

type LedgerTaskProcessOutcome string

const (
	LedgerTaskProcessIdle       LedgerTaskProcessOutcome = "IDLE"
	LedgerTaskProcessCompleted  LedgerTaskProcessOutcome = "COMPLETED"
	LedgerTaskProcessRetrying   LedgerTaskProcessOutcome = "RETRYING"
	LedgerTaskProcessDeadLetter LedgerTaskProcessOutcome = "DEAD_LETTER"
)

type LedgerTaskProcessResult struct {
	Outcome LedgerTaskProcessOutcome
	TaskID  int64
	Kind    LedgerTaskKind
}

func (r *DBRepository) ClaimCompanyFundLedgerTask(
	ctx context.Context,
	owner string,
	leaseDuration time.Duration,
) (CompanyFundLedgerTaskLease, bool, error) {
	if err := r.requireDB(); err != nil {
		return CompanyFundLedgerTaskLease{}, false, err
	}
	owner = strings.TrimSpace(owner)
	if owner == "" || len(owner) > 128 || leaseDuration <= 0 {
		return CompanyFundLedgerTaskLease{}, false, fmt.Errorf("invalid company-fund ledger task lease request")
	}
	var (
		lease         CompanyFundLedgerTaskLease
		channel       string
		kind          string
		subjectID     sql.NullInt64
		referenceType string
		payload       string
		payloadDigest string
	)
	err := r.db.QueryRowContext(ctx, claimCompanyFundLedgerTaskSQL, owner, leaseDuration.Microseconds()).Scan(
		&lease.ID,
		&channel,
		&lease.ProviderAccountKey,
		&kind,
		&lease.ProviderTransactionFactID,
		&lease.SubjectProviderTransactionID,
		&subjectID,
		&referenceType,
		&lease.RelationshipReferenceKey,
		&lease.RelationshipGroupKey,
		&lease.EvidenceReference,
		&payload,
		&payloadDigest,
		&lease.PolicyVersion,
		&lease.AttemptCount,
		&lease.SLAExpiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return CompanyFundLedgerTaskLease{}, false, nil
	}
	if err != nil {
		return CompanyFundLedgerTaskLease{}, false, fmt.Errorf("claim company-fund ledger task: %w", err)
	}
	if payloadSHA256Hex([]byte(payload)) != payloadDigest {
		return CompanyFundLedgerTaskLease{}, false, fmt.Errorf("company-fund ledger task payload digest mismatch")
	}
	lease.Channel = TransactionSource(channel)
	lease.Kind = LedgerTaskKind(kind)
	lease.RelationshipReferenceType = RelationshipReferenceType(referenceType)
	if subjectID.Valid {
		value := subjectID.Int64
		lease.SubjectTransactionID = &value
	}
	lease.Proposal, err = decodeCompanyFundLedgerTaskPayload([]byte(payload))
	if err != nil {
		return CompanyFundLedgerTaskLease{}, false, err
	}
	return lease, true, nil
}

func (r *DBRepository) completeCompanyFundLedgerTask(
	ctx context.Context,
	taskID int64,
	owner string,
	attemptCount int,
	subjectTransactionID *int64,
) error {
	var updated int64
	if err := r.db.QueryRowContext(ctx, completeCompanyFundLedgerTaskSQL,
		taskID, owner, nullableInt64(subjectTransactionID), attemptCount,
	).Scan(&updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("company-fund ledger task lease was lost")
		}
		return fmt.Errorf("complete company-fund ledger task: %w", err)
	}
	return nil
}

func (r *DBRepository) retryCompanyFundLedgerTask(
	ctx context.Context,
	taskID int64,
	owner string,
	attemptCount int,
	retryAfter time.Duration,
	errorCode string,
) (LedgerTaskProcessOutcome, error) {
	if retryAfter <= 0 {
		retryAfter = time.Minute
	}
	var state string
	if err := r.db.QueryRowContext(ctx, retryCompanyFundLedgerTaskSQL,
		taskID, owner, retryAfter.Microseconds(), safeLedgerTaskErrorCode(errorCode), attemptCount,
	).Scan(&state); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("company-fund ledger task lease was lost")
		}
		return "", fmt.Errorf("retry company-fund ledger task: %w", err)
	}
	if LedgerTaskState(state) == LedgerTaskStateDeadLetter {
		return LedgerTaskProcessDeadLetter, nil
	}
	return LedgerTaskProcessRetrying, nil
}

func (r *DBRepository) deadLetterCompanyFundLedgerTask(
	ctx context.Context,
	taskID int64,
	owner string,
	attemptCount int,
	errorCode string,
) error {
	var updated int64
	if err := r.db.QueryRowContext(ctx, deadLetterCompanyFundLedgerTaskSQL,
		taskID, owner, safeLedgerTaskErrorCode(errorCode), attemptCount,
	).Scan(&updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("company-fund ledger task lease was lost")
		}
		return fmt.Errorf("dead-letter company-fund ledger task: %w", err)
	}
	return nil
}

func (r *DBRepository) NextCompanyFundLedgerTaskDue(ctx context.Context) (time.Time, error) {
	if err := r.requireDB(); err != nil {
		return time.Time{}, err
	}
	var due sql.NullTime
	if err := r.db.QueryRowContext(ctx, nextCompanyFundLedgerTaskDueSQL).Scan(&due); err != nil {
		return time.Time{}, fmt.Errorf("query next company-fund ledger task due: %w", err)
	}
	if !due.Valid {
		return time.Time{}, nil
	}
	return due.Time.UTC(), nil
}

var _ interface {
	ClaimCompanyFundLedgerTask(context.Context, string, time.Duration) (CompanyFundLedgerTaskLease, bool, error)
} = (*DBRepository)(nil)
