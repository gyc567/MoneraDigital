package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"monera-digital/internal/migration"
)

// AddSafeheronRoutingStatusChecks adds the durable provider-status fallback
// queue used when a transaction webhook stops at a non-terminal status.
type AddSafeheronRoutingStatusChecks struct{}

func (*AddSafeheronRoutingStatusChecks) Version() string { return "064" }

func (*AddSafeheronRoutingStatusChecks) Description() string {
	return "Add durable Safeheron non-terminal transaction status checks"
}

func (*AddSafeheronRoutingStatusChecks) RequiredPreexistingVersion() string { return "063" }

func (*AddSafeheronRoutingStatusChecks) RequiredExpectedCeiling() string { return "064" }

func (*AddSafeheronRoutingStatusChecks) Up(*sql.DB) error {
	return fmt.Errorf("064 is controlled; run it through Migrator.MigrateWithExpectedCeiling")
}

func (*AddSafeheronRoutingStatusChecks) UpTx(tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("migration 064 transaction is required")
	}
	ctx := context.Background()
	if _, err := tx.ExecContext(ctx, migration064TimeoutsSQL); err != nil {
		return fmt.Errorf("configure migration 064 timeouts: %w", err)
	}
	if _, err := tx.ExecContext(ctx, migration064SchemaSQL); err != nil {
		return fmt.Errorf("install Safeheron routing status checks: %w", err)
	}
	return nil
}

func (*AddSafeheronRoutingStatusChecks) Down(*sql.DB) error {
	return fmt.Errorf("064 is forward-only; Safeheron routing status checks must be changed by a new migration")
}

var _ migration.Migration = (*AddSafeheronRoutingStatusChecks)(nil)
var _ migration.ControlledMigration = (*AddSafeheronRoutingStatusChecks)(nil)

const migration064TimeoutsSQL = `SET LOCAL search_path = pg_catalog, public; SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '60s'; SET LOCAL idle_in_transaction_session_timeout = '60s';`

const migration064SchemaSQL = `
UPDATE public.safeheron_transaction_routing_alerts
SET transition_key=CASE
  WHEN payload->>'reason_code'='STATUS_NOT_TERMINAL'
    THEN 'sla:pending:level:' || (payload->>'level')
  ELSE 'sla:open:level:' || (payload->>'level')
END
WHERE alert_type='SLA_ESCALATION'
  AND transition_key LIKE 'sla:level:%';

CREATE TABLE public.safeheron_transaction_routing_status_checks (
  safeheron_tx_key VARCHAR(128) PRIMARY KEY,
  first_seen_at TIMESTAMPTZ NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  next_check_at TIMESTAMPTZ,
  last_checked_at TIMESTAMPTZ,
  last_check_outcome VARCHAR(16) CHECK (last_check_outcome IN ('OBSERVED', 'ERROR')),
  last_observed_status VARCHAR(64),
  last_provider_event_id VARCHAR(128),
  last_error_code VARCHAR(64),
  lease_owner VARCHAR(128),
  lease_expires_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK ((last_checked_at IS NULL) = (last_check_outcome IS NULL)),
  CHECK (last_provider_event_id IS NULL OR last_check_outcome = 'OBSERVED'),
  CHECK (last_error_code IS NULL OR last_check_outcome = 'ERROR'),
  CHECK ((lease_owner IS NULL) = (lease_expires_at IS NULL)),
  CHECK (
    (completed_at IS NULL AND next_check_at IS NOT NULL)
    OR (completed_at IS NOT NULL AND next_check_at IS NULL AND lease_owner IS NULL)
  )
);

CREATE INDEX idx_safeheron_routing_status_checks_claim
  ON public.safeheron_transaction_routing_status_checks (next_check_at, lease_expires_at, safeheron_tx_key)
  WHERE completed_at IS NULL;
`
