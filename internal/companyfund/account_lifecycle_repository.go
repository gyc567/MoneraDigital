package companyfund

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const claimAccountLifecycleCommandSQL = `
WITH candidate AS (
  SELECT id
  FROM company_fund_account_lifecycle_commands
  WHERE (status = 'PENDING' AND next_attempt_at <= clock_timestamp())
     OR (status = 'PROCESSING' AND lease_expires_at <= clock_timestamp())
  ORDER BY next_attempt_at, id
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE company_fund_account_lifecycle_commands command
SET status = 'PROCESSING',
    attempt_count = attempt_count + 1,
    lease_owner = $1,
    lease_expires_at = clock_timestamp() + ($2::bigint * INTERVAL '1 microsecond'),
    started_at = COALESCE(started_at, clock_timestamp()),
    updated_at = clock_timestamp()
FROM candidate, company_fund_accounts target
WHERE command.id = candidate.id
  AND target.id = command.target_account_id
RETURNING command.id,
  command.command_type,
  command.target_account_id,
  command.related_account_id,
  target.provider_account_key,
  command.requested_provider_account_key,
  command.requested_by,
  command.reason,
  command.expected_target_version,
  command.expected_related_version,
  command.attempt_count,
  command.lease_owner,
  command.lease_expires_at,
  command.business_applied_at IS NOT NULL`

const lockAccountLifecycleCommandSQL = `
SELECT command_type, target_account_id, COALESCE(related_account_id, 0),
       COALESCE(requested_provider_account_key, ''), requested_by, reason,
       expected_target_version, COALESCE(expected_related_version, 0),
       business_applied_at IS NOT NULL
FROM company_fund_account_lifecycle_commands
WHERE id = $1
  AND status = 'PROCESSING'
  AND lease_owner = $2
  AND attempt_count = $3
  AND lease_expires_at > clock_timestamp()
FOR UPDATE`

const lockAirwallexLifecycleAccountSQL = `
SELECT id, provider_account_key, airwallex_lifecycle, lifecycle_version,
       is_enabled, first_enabled_at
FROM company_fund_accounts
WHERE id = $1 AND channel = 'AIRWALLEX'
FOR UPDATE`

const setAccountLifecycleCommandGuardSQL = `
SET LOCAL monera.airwallex_account_lifecycle_command = 'on';
SELECT pg_advisory_xact_lock(768972734063)`

const validateAirwallexCandidateSQL = `
UPDATE company_fund_accounts
SET airwallex_validated_at = clock_timestamp(),
    airwallex_provider_identity_summary = $2::jsonb,
    lifecycle_version = lifecycle_version + 1,
    updated_at = clock_timestamp()
WHERE id = $1`

const pauseAirwallexAccountSQL = `
UPDATE company_fund_accounts
SET airwallex_lifecycle = 'PAUSED',
    is_enabled = false,
    lifecycle_version = lifecycle_version + 1,
    updated_at = clock_timestamp()
WHERE id = $1`

const resumeAirwallexAccountSQL = `
UPDATE company_fund_accounts
SET airwallex_lifecycle = 'CURRENT',
    is_enabled = true,
    airwallex_validated_at = clock_timestamp(),
    airwallex_provider_identity_summary = $2::jsonb,
    lifecycle_version = lifecycle_version + 1,
    updated_at = clock_timestamp()
WHERE id = $1`

// This manifest is intentionally explicit. Adding another mutable
// provider_account_key projection requires adding it here and to the schema
// coverage check before identity correction may ship.
const correctAirwallexProviderAccountReferencesSQL = `
UPDATE company_fund_provider_events
SET provider_account_key = $2, updated_at = clock_timestamp()
WHERE channel = 'AIRWALLEX' AND provider_account_key = $1;
UPDATE company_fund_provider_transaction_facts
SET provider_account_key = $2, updated_at = clock_timestamp()
WHERE channel = 'AIRWALLEX' AND provider_account_key = $1;
UPDATE company_fund_transactions
SET provider_account_key = $2, updated_at = clock_timestamp()
WHERE channel = 'AIRWALLEX' AND provider_account_key = $1;
UPDATE company_fund_ledger_tasks
SET provider_account_key = $2, updated_at = clock_timestamp()
WHERE channel = 'AIRWALLEX' AND provider_account_key = $1`

var correctAirwallexProviderAccountReferenceStatements = []string{
	`UPDATE company_fund_provider_events
SET provider_account_key = $2, updated_at = clock_timestamp()
WHERE channel = 'AIRWALLEX' AND provider_account_key = $1`,
	`UPDATE company_fund_provider_transaction_facts
SET provider_account_key = $2, updated_at = clock_timestamp()
WHERE channel = 'AIRWALLEX' AND provider_account_key = $1`,
	`UPDATE company_fund_transactions
SET provider_account_key = $2, updated_at = clock_timestamp()
WHERE channel = 'AIRWALLEX' AND provider_account_key = $1`,
	`UPDATE company_fund_ledger_tasks
SET provider_account_key = $2, updated_at = clock_timestamp()
WHERE channel = 'AIRWALLEX' AND provider_account_key = $1`,
}

const correctAirwallexAccountIdentitySQL = `
UPDATE company_fund_accounts
SET provider_account_key = $2,
    airwallex_validated_at = clock_timestamp(),
    airwallex_provider_identity_summary = $3::jsonb,
    lifecycle_version = lifecycle_version + 1,
    updated_at = clock_timestamp()
WHERE id = $1`

const retireAirwallexAccountSQL = `
UPDATE company_fund_accounts
SET airwallex_lifecycle = 'RETIRED',
    is_enabled = false,
    lifecycle_version = lifecycle_version + 1,
    updated_at = clock_timestamp()
WHERE id = $1`

const countUnfinishedAirwallexProviderEventsSQL = `
SELECT count(*)
FROM company_fund_provider_events
WHERE channel = 'AIRWALLEX'
  AND provider_account_key = $1
  AND event_state IN ('PENDING', 'LEASED', 'FAILED')`

const activateAirwallexCandidateSQL = `
UPDATE company_fund_accounts
SET airwallex_lifecycle = 'CURRENT',
    is_enabled = true,
    first_enabled_at = COALESCE(first_enabled_at, clock_timestamp()),
    airwallex_validated_at = clock_timestamp(),
    airwallex_provider_identity_summary = $2::jsonb,
    lifecycle_version = lifecycle_version + 1,
    updated_at = clock_timestamp()
WHERE id = $1`

const countAirwallexCandidateReferencesSQL = `
SELECT
  (SELECT count(*) FROM company_fund_account_asset_policies
    WHERE company_fund_account_id = $1)
  + (SELECT count(*) FROM company_fund_provider_events
    WHERE channel = 'AIRWALLEX' AND provider_account_key = $2)
  + (SELECT count(*) FROM company_fund_provider_transaction_facts
    WHERE channel = 'AIRWALLEX' AND provider_account_key = $2)
  + (SELECT count(*) FROM company_fund_transactions
    WHERE channel = 'AIRWALLEX'
      AND (provider_account_key = $2
        OR from_company_fund_account_id = $1
        OR to_company_fund_account_id = $1))
  + (SELECT count(*) FROM company_fund_ledger_tasks
    WHERE channel = 'AIRWALLEX' AND provider_account_key = $2)
  + (SELECT count(*) FROM company_fund_account_lifecycle_commands
    WHERE id <> $3
      AND status IN ('PENDING', 'PROCESSING')
      AND (target_account_id = $1 OR related_account_id = $1))`

const deleteAirwallexCandidateSQL = `
UPDATE company_fund_accounts
SET airwallex_lifecycle = 'DELETED',
    is_enabled = false,
    deleted_at = clock_timestamp(),
    deleted_by = $2,
    delete_reason = $3,
    lifecycle_version = lifecycle_version + 1,
    updated_at = clock_timestamp()
WHERE id = $1`

const insertAccountLifecycleAuditSQL = `
INSERT INTO company_fund_account_lifecycle_audits (
  command_id, account_id, command_type, old_lifecycle, new_lifecycle,
  old_provider_account_key, new_provider_account_key, actor, reason
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (command_id, account_id) DO NOTHING`

const markAccountLifecycleBusinessAppliedSQL = `
UPDATE company_fund_account_lifecycle_commands
SET business_applied_at = COALESCE(business_applied_at, clock_timestamp()),
    result_summary = $4::jsonb,
    error_code = NULL,
    error_message = NULL,
    updated_at = clock_timestamp()
WHERE id = $1
  AND status = 'PROCESSING'
  AND lease_owner = $2
  AND attempt_count = $3
  AND lease_expires_at > clock_timestamp()`

const completeAccountLifecycleCommandSQL = `
UPDATE company_fund_account_lifecycle_commands
SET status = 'SUCCEEDED',
    lease_owner = NULL,
    lease_expires_at = NULL,
    result_summary = jsonb_set(
      COALESCE(result_summary, '{}'::jsonb),
      '{registryPublished}',
      'true'::jsonb,
      true
    ),
    completed_at = clock_timestamp(),
    updated_at = clock_timestamp()
WHERE id = $1
  AND status = 'PROCESSING'
  AND lease_owner = $2
  AND attempt_count = $3
  AND business_applied_at IS NOT NULL
  AND lease_expires_at > clock_timestamp()`

const failAccountLifecycleCommandSQL = `
UPDATE company_fund_account_lifecycle_commands
SET status = 'FAILED',
    lease_owner = NULL,
    lease_expires_at = NULL,
    error_code = $4,
    error_message = $5,
    completed_at = clock_timestamp(),
    updated_at = clock_timestamp()
WHERE id = $1
  AND status = 'PROCESSING'
  AND lease_owner = $2
  AND attempt_count = $3
  AND business_applied_at IS NULL
  AND lease_expires_at > clock_timestamp()`

const retryAccountLifecycleCommandSQL = `
UPDATE company_fund_account_lifecycle_commands
SET status = 'PENDING',
    lease_owner = NULL,
    lease_expires_at = NULL,
    next_attempt_at = clock_timestamp() + ($4::bigint * INTERVAL '1 microsecond'),
    error_code = $5,
    error_message = $6,
    updated_at = clock_timestamp()
WHERE id = $1
  AND status = 'PROCESSING'
  AND lease_owner = $2
  AND attempt_count = $3
  AND (
    ($7::boolean AND business_applied_at IS NOT NULL)
    OR (NOT $7::boolean AND business_applied_at IS NULL)
  )
  AND lease_expires_at > clock_timestamp()`

const nextAccountLifecycleCommandDueSQL = `
SELECT MIN(
  CASE WHEN status = 'PENDING' THEN next_attempt_at ELSE lease_expires_at END
)
FROM company_fund_account_lifecycle_commands
WHERE status IN ('PENDING', 'PROCESSING')`

type lockedAirwallexLifecycleAccount struct {
	ID                 int64
	ProviderAccountKey string
	Lifecycle          AirwallexAccountLifecycle
	Version            int64
	Enabled            bool
	FirstEnabledAt     sql.NullTime
}

func (r *DBRepository) ClaimAccountLifecycleCommand(
	ctx context.Context,
	owner string,
	leaseDuration time.Duration,
) (AccountLifecycleCommandLease, bool, error) {
	if err := r.requireDB(); err != nil {
		return AccountLifecycleCommandLease{}, false, err
	}
	owner = strings.TrimSpace(owner)
	if owner == "" || len(owner) > 128 || leaseDuration <= 0 {
		return AccountLifecycleCommandLease{}, false, fmt.Errorf("invalid account lifecycle command lease")
	}
	var (
		lease        AccountLifecycleCommandLease
		commandType  string
		relatedID    sql.NullInt64
		requestedKey sql.NullString
		relatedVer   sql.NullInt64
	)
	err := r.db.QueryRowContext(
		ctx,
		claimAccountLifecycleCommandSQL,
		owner,
		leaseDuration.Microseconds(),
	).Scan(
		&lease.ID,
		&commandType,
		&lease.TargetAccountID,
		&relatedID,
		&lease.TargetProviderKey,
		&requestedKey,
		&lease.RequestedBy,
		&lease.Reason,
		&lease.ExpectedTargetVer,
		&relatedVer,
		&lease.AttemptCount,
		&lease.LeaseOwner,
		&lease.LeaseExpiresAt,
		&lease.BusinessApplied,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AccountLifecycleCommandLease{}, false, nil
	}
	if err != nil {
		return AccountLifecycleCommandLease{}, false, fmt.Errorf("claim account lifecycle command: %w", err)
	}
	lease.Type = AccountLifecycleCommandType(commandType)
	if relatedID.Valid {
		lease.RelatedAccountID = relatedID.Int64
	}
	if requestedKey.Valid {
		lease.RequestedProviderKey = requestedKey.String
	}
	if relatedVer.Valid {
		lease.ExpectedRelatedVer = relatedVer.Int64
	}
	return lease, true, nil
}

func (r *DBRepository) ApplyAccountLifecycleCommand(
	ctx context.Context,
	input AccountLifecycleApplyInput,
) error {
	if err := r.requireDB(); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin account lifecycle command: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, setAccountLifecycleCommandGuardSQL); err != nil {
		return fmt.Errorf("enable account lifecycle command guard: %w", err)
	}
	lockedLease, err := lockAccountLifecycleCommand(ctx, tx, input.Lease)
	if err != nil {
		return err
	}
	if lockedLease.BusinessApplied {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit recovered account lifecycle command: %w", err)
		}
		committed = true
		return nil
	}
	target, err := lockAirwallexLifecycleAccount(ctx, tx, lockedLease.TargetAccountID)
	if err != nil {
		return err
	}
	if target.Version != lockedLease.ExpectedTargetVer {
		return fmt.Errorf("target account lifecycle version conflict")
	}
	if err := validateAccountLifecycleCommandSource(lockedLease.Type, target.Lifecycle); err != nil {
		return err
	}
	if err := applyLockedAccountLifecycleCommand(ctx, tx, lockedLease, target, input.ProviderIdentity); err != nil {
		return err
	}
	resultSummary, err := json.Marshal(map[string]any{
		"accountId":          target.ID,
		"commandType":        lockedLease.Type,
		"providerValidated":  input.ProviderIdentity.ProviderAccountID != "",
		"registryPublished":  false,
		"businessStateReady": true,
	})
	if err != nil {
		return fmt.Errorf("encode lifecycle command result: %w", err)
	}
	result, err := tx.ExecContext(
		ctx,
		markAccountLifecycleBusinessAppliedSQL,
		lockedLease.ID,
		lockedLease.LeaseOwner,
		lockedLease.AttemptCount,
		string(resultSummary),
	)
	if err != nil {
		return fmt.Errorf("mark lifecycle command business applied: %w", err)
	}
	if count, countErr := result.RowsAffected(); countErr != nil || count != 1 {
		return fmt.Errorf("account lifecycle command lease was lost before commit")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit account lifecycle command: %w", err)
	}
	committed = true
	return nil
}

func lockAccountLifecycleCommand(
	ctx context.Context,
	tx *sql.Tx,
	claimed AccountLifecycleCommandLease,
) (AccountLifecycleCommandLease, error) {
	var (
		locked       AccountLifecycleCommandLease
		commandType  string
		requestedKey string
	)
	locked.ID = claimed.ID
	locked.LeaseOwner = claimed.LeaseOwner
	locked.AttemptCount = claimed.AttemptCount
	if err := tx.QueryRowContext(
		ctx,
		lockAccountLifecycleCommandSQL,
		claimed.ID,
		claimed.LeaseOwner,
		claimed.AttemptCount,
	).Scan(
		&commandType,
		&locked.TargetAccountID,
		&locked.RelatedAccountID,
		&requestedKey,
		&locked.RequestedBy,
		&locked.Reason,
		&locked.ExpectedTargetVer,
		&locked.ExpectedRelatedVer,
		&locked.BusinessApplied,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AccountLifecycleCommandLease{}, fmt.Errorf("account lifecycle command lease was lost")
		}
		return AccountLifecycleCommandLease{}, fmt.Errorf("lock account lifecycle command: %w", err)
	}
	locked.Type = AccountLifecycleCommandType(commandType)
	locked.RequestedProviderKey = requestedKey
	return locked, nil
}

func lockAirwallexLifecycleAccount(
	ctx context.Context,
	tx *sql.Tx,
	accountID int64,
) (lockedAirwallexLifecycleAccount, error) {
	var account lockedAirwallexLifecycleAccount
	var lifecycle string
	if err := tx.QueryRowContext(ctx, lockAirwallexLifecycleAccountSQL, accountID).Scan(
		&account.ID,
		&account.ProviderAccountKey,
		&lifecycle,
		&account.Version,
		&account.Enabled,
		&account.FirstEnabledAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return account, fmt.Errorf("Airwallex lifecycle account does not exist")
		}
		return account, fmt.Errorf("lock Airwallex lifecycle account: %w", err)
	}
	account.Lifecycle = AirwallexAccountLifecycle(lifecycle)
	return account, nil
}

func validateAccountLifecycleCommandSource(
	command AccountLifecycleCommandType,
	source AirwallexAccountLifecycle,
) error {
	valid := false
	switch command {
	case AccountLifecycleCommandValidateCandidate,
		AccountLifecycleCommandCutover,
		AccountLifecycleCommandDeleteCandidate:
		valid = source == AirwallexLifecycleCandidate
	case AccountLifecycleCommandPause:
		valid = source == AirwallexLifecycleCurrent
	case AccountLifecycleCommandResume:
		valid = source == AirwallexLifecyclePaused
	case AccountLifecycleCommandCorrectIdentity:
		valid = source == AirwallexLifecycleCandidate ||
			source == AirwallexLifecycleCurrent ||
			source == AirwallexLifecyclePaused
	}
	if !valid {
		return fmt.Errorf("account lifecycle %s does not permit command %s", source, command)
	}
	return nil
}

func applyLockedAccountLifecycleCommand(
	ctx context.Context,
	tx *sql.Tx,
	lease AccountLifecycleCommandLease,
	target lockedAirwallexLifecycleAccount,
	identity AirwallexProviderIdentitySummary,
) error {
	identityJSON, err := json.Marshal(identity)
	if err != nil {
		return fmt.Errorf("encode provider identity summary: %w", err)
	}
	newLifecycle := target.Lifecycle
	newProviderKey := target.ProviderAccountKey

	switch lease.Type {
	case AccountLifecycleCommandValidateCandidate:
		if err := requireValidatedProviderIdentity(identity, target.ProviderAccountKey); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, validateAirwallexCandidateSQL, target.ID, string(identityJSON))
	case AccountLifecycleCommandPause:
		newLifecycle = AirwallexLifecyclePaused
		_, err = tx.ExecContext(ctx, pauseAirwallexAccountSQL, target.ID)
	case AccountLifecycleCommandResume:
		if err := requireValidatedProviderIdentity(identity, target.ProviderAccountKey); err != nil {
			return err
		}
		newLifecycle = AirwallexLifecycleCurrent
		_, err = tx.ExecContext(ctx, resumeAirwallexAccountSQL, target.ID, string(identityJSON))
	case AccountLifecycleCommandCorrectIdentity:
		if err := requireValidatedProviderIdentity(identity, lease.RequestedProviderKey); err != nil {
			return err
		}
		if lease.RequestedProviderKey == target.ProviderAccountKey {
			return fmt.Errorf("corrected Airwallex provider account key must differ from the current key")
		}
		for _, statement := range correctAirwallexProviderAccountReferenceStatements {
			if _, err = tx.ExecContext(ctx, statement, target.ProviderAccountKey, lease.RequestedProviderKey); err != nil {
				return fmt.Errorf("repair Airwallex provider account references: %w", err)
			}
		}
		newProviderKey = lease.RequestedProviderKey
		_, err = tx.ExecContext(
			ctx,
			correctAirwallexAccountIdentitySQL,
			target.ID,
			lease.RequestedProviderKey,
			string(identityJSON),
		)
	case AccountLifecycleCommandCutover:
		if err := requireValidatedProviderIdentity(identity, target.ProviderAccountKey); err != nil {
			return err
		}
		if lease.RelatedAccountID != 0 {
			var related lockedAirwallexLifecycleAccount
			related, err = lockAirwallexLifecycleAccount(ctx, tx, lease.RelatedAccountID)
			if err != nil {
				return err
			}
			if related.Version != lease.ExpectedRelatedVer ||
				(related.Lifecycle != AirwallexLifecycleCurrent &&
					related.Lifecycle != AirwallexLifecyclePaused) {
				return fmt.Errorf("related Airwallex account state or version conflict")
			}
			var unfinishedEvents int64
			if err = tx.QueryRowContext(
				ctx,
				countUnfinishedAirwallexProviderEventsSQL,
				related.ProviderAccountKey,
			).Scan(&unfinishedEvents); err != nil {
				return fmt.Errorf("check prior Airwallex account event queue: %w", err)
			}
			if unfinishedEvents != 0 {
				return fmt.Errorf(
					"%w: prior account has %d",
					ErrAccountLifecycleUnfinishedProviderEvents,
					unfinishedEvents,
				)
			}
			if _, err = tx.ExecContext(ctx, retireAirwallexAccountSQL, related.ID); err != nil {
				return fmt.Errorf("retire prior Airwallex account: %w", err)
			}
			if err = insertAccountLifecycleAudit(
				ctx,
				tx,
				lease,
				related,
				AirwallexLifecycleRetired,
				related.ProviderAccountKey,
			); err != nil {
				return err
			}
		}
		newLifecycle = AirwallexLifecycleCurrent
		_, err = tx.ExecContext(ctx, activateAirwallexCandidateSQL, target.ID, string(identityJSON))
	case AccountLifecycleCommandDeleteCandidate:
		if target.FirstEnabledAt.Valid {
			return fmt.Errorf("previously enabled Airwallex candidate cannot be deleted")
		}
		var references int64
		if err = tx.QueryRowContext(
			ctx,
			countAirwallexCandidateReferencesSQL,
			target.ID,
			target.ProviderAccountKey,
			lease.ID,
		).Scan(&references); err != nil {
			return fmt.Errorf("check Airwallex candidate references: %w", err)
		}
		if references != 0 {
			return fmt.Errorf("Airwallex candidate has %d business references", references)
		}
		newLifecycle = AirwallexLifecycleDeleted
		_, err = tx.ExecContext(
			ctx,
			deleteAirwallexCandidateSQL,
			target.ID,
			lease.RequestedBy,
			lease.Reason,
		)
	default:
		return fmt.Errorf("unsupported account lifecycle command")
	}
	if err != nil {
		return fmt.Errorf("apply account lifecycle transition: %w", err)
	}
	return insertAccountLifecycleAudit(
		ctx,
		tx,
		lease,
		target,
		newLifecycle,
		newProviderKey,
	)
}

func requireValidatedProviderIdentity(
	identity AirwallexProviderIdentitySummary,
	expectedProviderKey string,
) error {
	if expectedProviderKey == "" || expectedProviderKey != strings.TrimSpace(expectedProviderKey) ||
		identity.ProviderAccountID != expectedProviderKey {
		return fmt.Errorf("validated Airwallex identity does not match the exact account key")
	}
	return nil
}

func insertAccountLifecycleAudit(
	ctx context.Context,
	tx *sql.Tx,
	lease AccountLifecycleCommandLease,
	account lockedAirwallexLifecycleAccount,
	newLifecycle AirwallexAccountLifecycle,
	newProviderKey string,
) error {
	if _, err := tx.ExecContext(
		ctx,
		insertAccountLifecycleAuditSQL,
		lease.ID,
		account.ID,
		lease.Type,
		account.Lifecycle,
		newLifecycle,
		account.ProviderAccountKey,
		newProviderKey,
		lease.RequestedBy,
		lease.Reason,
	); err != nil {
		return fmt.Errorf("write account lifecycle audit: %w", err)
	}
	return nil
}

func (r *DBRepository) CompleteAccountLifecycleCommand(
	ctx context.Context,
	commandID int64,
	owner string,
	attemptCount int,
) error {
	return execExactlyOne(
		ctx,
		r.db,
		completeAccountLifecycleCommandSQL,
		"complete account lifecycle command",
		commandID,
		owner,
		attemptCount,
	)
}

func (r *DBRepository) FailAccountLifecycleCommand(
	ctx context.Context,
	failure AccountLifecycleFailure,
) error {
	return execExactlyOne(
		ctx,
		r.db,
		failAccountLifecycleCommandSQL,
		"fail account lifecycle command",
		failure.CommandID,
		failure.LeaseOwner,
		failure.AttemptCount,
		safeAccountLifecycleErrorCode(failure.ErrorCode),
		safeAccountLifecycleMessage(failure.SafeMessage),
	)
}

func (r *DBRepository) RetryAccountLifecycleCommand(
	ctx context.Context,
	failure AccountLifecycleFailure,
) error {
	if failure.RetryAfter <= 0 {
		return fmt.Errorf("invalid account lifecycle retry")
	}
	return execExactlyOne(
		ctx,
		r.db,
		retryAccountLifecycleCommandSQL,
		"retry account lifecycle command",
		failure.CommandID,
		failure.LeaseOwner,
		failure.AttemptCount,
		failure.RetryAfter.Microseconds(),
		safeAccountLifecycleErrorCode(failure.ErrorCode),
		safeAccountLifecycleMessage(failure.SafeMessage),
		failure.BusinessApplied,
	)
}

func (r *DBRepository) NextAccountLifecycleCommandDue(
	ctx context.Context,
) (time.Time, error) {
	var due sql.NullTime
	if err := r.db.QueryRowContext(ctx, nextAccountLifecycleCommandDueSQL).Scan(&due); err != nil {
		return time.Time{}, fmt.Errorf("load next account lifecycle command due: %w", err)
	}
	if !due.Valid {
		return time.Time{}, nil
	}
	return due.Time.UTC(), nil
}

type contextExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func execExactlyOne(
	ctx context.Context,
	execer contextExecer,
	query string,
	operation string,
	args ...any,
) error {
	result, err := execer.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return fmt.Errorf("%s: command lease was lost", operation)
	}
	return nil
}

func safeAccountLifecycleErrorCode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 64 {
		return "LIFECYCLE_COMMAND_FAILED"
	}
	return value
}

func safeAccountLifecycleMessage(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Account lifecycle command failed"
	}
	if len(value) > 1000 {
		return value[:1000]
	}
	return value
}

var _ AccountLifecycleCommandRepository = (*DBRepository)(nil)
