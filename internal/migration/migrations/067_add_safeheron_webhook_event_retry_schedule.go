package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"monera-digital/internal/migration"
)

// AddSafeheronWebhookEventRetrySchedule prevents a deferred AML event from
// monopolizing the head of the raw webhook queue while ownership settles.
type AddSafeheronWebhookEventRetrySchedule struct{}

func (*AddSafeheronWebhookEventRetrySchedule) Version() string { return "067" }

func (*AddSafeheronWebhookEventRetrySchedule) Description() string {
	return "Add durable Safeheron webhook event retry scheduling"
}

func (*AddSafeheronWebhookEventRetrySchedule) RequiredPreexistingVersion() string { return "066" }

func (*AddSafeheronWebhookEventRetrySchedule) RequiredExpectedCeiling() string { return "067" }

func (*AddSafeheronWebhookEventRetrySchedule) Up(*sql.DB) error {
	return fmt.Errorf("067 is controlled; run it through Migrator.MigrateWithExpectedCeiling")
}

func (*AddSafeheronWebhookEventRetrySchedule) UpTx(tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("migration 067 transaction is required")
	}
	ctx := context.Background()
	if _, err := tx.ExecContext(ctx, migration067TimeoutsSQL); err != nil {
		return fmt.Errorf("configure migration 067 timeouts: %w", err)
	}
	if _, err := tx.ExecContext(ctx, migration067SchemaSQL); err != nil {
		return fmt.Errorf("install Safeheron webhook event retry schedule: %w", err)
	}
	return nil
}

func (*AddSafeheronWebhookEventRetrySchedule) Down(*sql.DB) error {
	return fmt.Errorf("067 is forward-only; Safeheron webhook event retries must be changed by a new migration")
}

var _ migration.Migration = (*AddSafeheronWebhookEventRetrySchedule)(nil)
var _ migration.ControlledMigration = (*AddSafeheronWebhookEventRetrySchedule)(nil)

const migration067TimeoutsSQL = `SET LOCAL search_path = pg_catalog, public; SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '60s'; SET LOCAL idle_in_transaction_session_timeout = '60s';`

const migration067SchemaSQL = `
ALTER TABLE public.safeheron_webhook_events
  ADD COLUMN next_attempt_at TIMESTAMPTZ;

ALTER TABLE public.safeheron_webhook_events
  ADD CONSTRAINT safeheron_webhook_events_next_attempt_state_check
  CHECK (process_status = 'PENDING' OR next_attempt_at IS NULL);

CREATE INDEX idx_safeheron_webhook_events_pending_retry
  ON public.safeheron_webhook_events (next_attempt_at, received_at, id)
  WHERE process_status = 'PENDING';
`
