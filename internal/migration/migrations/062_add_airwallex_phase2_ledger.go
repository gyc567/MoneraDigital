package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"monera-digital/internal/migration"
)

// AddAirwallexPhase2Ledger adds the durable coordination and finance ownership
// contract used by Airwallex FEE, CONVERSION, and REVERSAL processing.
//
// The DDL is intentionally strict rather than IF NOT EXISTS based. This
// ControlledMigration runs all statements and its provenance insert in one
// transaction, so an interrupted attempt rolls back atomically; pre-existing
// partial objects indicate schema drift and must fail closed for inspection.
type AddAirwallexPhase2Ledger struct{}

func (*AddAirwallexPhase2Ledger) Version() string { return "062" }

func (*AddAirwallexPhase2Ledger) Description() string {
	return "Add Airwallex Phase 2 relationship tasks and finance classification ownership"
}

func (*AddAirwallexPhase2Ledger) RequiredPreexistingVersion() string { return "061" }

func (*AddAirwallexPhase2Ledger) RequiredExpectedCeiling() string { return "062" }

func (*AddAirwallexPhase2Ledger) Up(*sql.DB) error {
	return fmt.Errorf("062 is controlled; run it through Migrator.MigrateWithExpectedCeiling")
}

func (*AddAirwallexPhase2Ledger) UpTx(tx *sql.Tx) error {
	ctx := context.Background()
	if _, err := tx.ExecContext(ctx, migration062TimeoutsSQL); err != nil {
		return fmt.Errorf("configure migration 062 timeouts: %w", err)
	}
	if _, err := tx.ExecContext(ctx, migration062SchemaSQL); err != nil {
		return fmt.Errorf("add Airwallex Phase 2 ledger contract: %w", err)
	}
	return nil
}

func (*AddAirwallexPhase2Ledger) Down(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("migration 062 rollback database is required")
	}
	if _, err := db.ExecContext(context.Background(), migration062DownSQL); err != nil {
		return fmt.Errorf("remove Airwallex Phase 2 ledger contract: %w", err)
	}
	return nil
}

var _ migration.Migration = (*AddAirwallexPhase2Ledger)(nil)
var _ migration.ControlledMigration = (*AddAirwallexPhase2Ledger)(nil)

const migration062TimeoutsSQL = `SET LOCAL search_path = pg_catalog, public; SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '60s'; SET LOCAL idle_in_transaction_session_timeout = '60s';`

const migration062SchemaSQL = `
ALTER TABLE public.company_fund_provider_transaction_facts
  ADD COLUMN provider_source_reference VARCHAR(256);

ALTER TABLE public.company_fund_transactions
  ADD COLUMN relationship_reference_type VARCHAR(32),
  ADD COLUMN relationship_reference_key VARCHAR(256),
  ADD COLUMN relationship_group_key VARCHAR(256),
  ADD COLUMN classification_source VARCHAR(32) NOT NULL DEFAULT 'UNCLASSIFIED',
  ADD COLUMN classification_policy_version VARCHAR(64);

ALTER TABLE public.company_fund_transactions
  ADD CONSTRAINT company_fund_transactions_relationship_reference_pair_check CHECK (
    (relationship_reference_type IS NULL AND relationship_reference_key IS NULL)
    OR (
      relationship_reference_type IN ('SOURCE_ID_EXACT_PARENT', 'SOURCE_ID_REVERSAL_TARGET', 'SOURCE_ID_CONVERSION_GROUP', 'SOURCE_ID_GROUP_ONLY', 'BATCH_ID_GROUP_ONLY')
      AND relationship_reference_key IS NOT NULL
      AND btrim(relationship_reference_key) <> ''
    )
  ),
  ADD CONSTRAINT company_fund_transactions_classification_source_check CHECK (
    classification_source IN ('UNCLASSIFIED', 'AUTO_RULE', 'INHERITED_REVERSAL', 'MANUAL')
  ),
  ADD CONSTRAINT company_fund_transactions_classification_policy_check CHECK (
    (
      classification_source IN ('AUTO_RULE', 'INHERITED_REVERSAL')
      AND classification_policy_version IS NOT NULL
      AND btrim(classification_policy_version) <> ''
    )
    OR (
      classification_source IN ('UNCLASSIFIED', 'MANUAL')
      AND classification_policy_version IS NULL
    )
  );

CREATE INDEX idx_company_fund_transactions_relationship_reference
  ON public.company_fund_transactions (
    channel,
    provider_account_key,
    relationship_reference_type,
    relationship_reference_key
  )
  WHERE relationship_reference_key IS NOT NULL;

CREATE INDEX idx_company_fund_transactions_relationship_group
  ON public.company_fund_transactions (channel, provider_account_key, relationship_group_key)
  WHERE relationship_group_key IS NOT NULL;

CREATE INDEX idx_company_fund_transactions_classification_maintenance
  ON public.company_fund_transactions (
    channel,
    movement_kind,
    classification_source,
    classification_policy_version,
    id
  );

CREATE TABLE public.company_fund_ledger_tasks (
  id BIGSERIAL PRIMARY KEY,
  channel VARCHAR(16) NOT NULL CHECK (channel = 'AIRWALLEX'),
  provider_account_key VARCHAR(128) NOT NULL CHECK (btrim(provider_account_key) <> ''),
  task_kind VARCHAR(32) NOT NULL CHECK (
    task_kind IN (
      'FEE_RELATIONSHIP',
      'CONVERSION_PAIR',
      'REVERSAL_RELATIONSHIP',
      'FEE_CLASSIFICATION',
      'REVERSAL_INHERITANCE'
    )
  ),
  task_state VARCHAR(16) NOT NULL DEFAULT 'WAITING' CHECK (
    task_state IN ('WAITING', 'LEASED', 'COMPLETED', 'DEAD_LETTER')
  ),
  provider_transaction_fact_id BIGINT NOT NULL
    REFERENCES public.company_fund_provider_transaction_facts(id) ON DELETE RESTRICT,
  subject_provider_transaction_id VARCHAR(256) NOT NULL
    CHECK (btrim(subject_provider_transaction_id) <> ''),
  subject_transaction_id BIGINT
    REFERENCES public.company_fund_transactions(id) ON DELETE RESTRICT,
  relationship_reference_type VARCHAR(32),
  relationship_reference_key VARCHAR(256),
  relationship_group_key VARCHAR(256),
  evidence_reference VARCHAR(512) NOT NULL CHECK (btrim(evidence_reference) <> ''),
  task_payload TEXT NOT NULL CHECK (jsonb_typeof(task_payload::jsonb) = 'object'),
  task_payload_digest VARCHAR(64) NOT NULL
    CHECK (task_payload_digest ~ '^[0-9a-f]{64}$'),
  policy_version VARCHAR(64),
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  lease_owner VARCHAR(128),
  lease_expires_at TIMESTAMPTZ,
  first_waiting_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  sla_expires_at TIMESTAMPTZ NOT NULL,
  terminal_at TIMESTAMPTZ,
  last_error_code VARCHAR(128),
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  CHECK (
    (task_state = 'LEASED' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
    OR (task_state <> 'LEASED' AND lease_owner IS NULL AND lease_expires_at IS NULL)
  ),
  CHECK (
    (task_state IN ('COMPLETED', 'DEAD_LETTER') AND terminal_at IS NOT NULL)
    OR (task_state NOT IN ('COMPLETED', 'DEAD_LETTER') AND terminal_at IS NULL)
  ),
  CHECK (
    (relationship_reference_type IS NULL AND relationship_reference_key IS NULL)
    OR (
      relationship_reference_type IN ('SOURCE_ID_EXACT_PARENT', 'SOURCE_ID_REVERSAL_TARGET', 'SOURCE_ID_CONVERSION_GROUP', 'SOURCE_ID_GROUP_ONLY', 'BATCH_ID_GROUP_ONLY')
      AND relationship_reference_key IS NOT NULL
      AND btrim(relationship_reference_key) <> ''
    )
  )
);

CREATE UNIQUE INDEX uq_company_fund_ledger_tasks_identity
  ON public.company_fund_ledger_tasks (
    channel,
    provider_account_key,
    task_kind,
    subject_provider_transaction_id,
    COALESCE(policy_version, '')
  );

CREATE INDEX idx_company_fund_ledger_tasks_due
  ON public.company_fund_ledger_tasks (task_state, next_attempt_at, lease_expires_at, id)
  WHERE task_state IN ('WAITING', 'LEASED');

CREATE INDEX idx_company_fund_ledger_tasks_relation
  ON public.company_fund_ledger_tasks (
    channel,
    provider_account_key,
    task_kind,
    relationship_group_key,
    relationship_reference_key
  );

CREATE OR REPLACE FUNCTION public.company_fund_mark_manual_classification()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  finance_fields_changed BOOLEAN;
BEGIN
  IF current_setting('monera.company_fund_classification_origin', true) = 'SYSTEM' THEN
    RETURN NEW;
  END IF;
  IF TG_OP = 'INSERT' THEN
    finance_fields_changed :=
      NEW.finance_category_level1_id IS NOT NULL
      OR NEW.finance_category_level2_id IS NOT NULL
      OR NEW.is_operating_income_expense IS NOT NULL
      OR NEW.summary_inclusion_override IS NOT NULL;
  ELSE
    finance_fields_changed :=
      NEW.finance_category_level1_id IS DISTINCT FROM OLD.finance_category_level1_id
      OR NEW.finance_category_level2_id IS DISTINCT FROM OLD.finance_category_level2_id
      OR NEW.is_operating_income_expense IS DISTINCT FROM OLD.is_operating_income_expense
      OR NEW.summary_inclusion_override IS DISTINCT FROM OLD.summary_inclusion_override;
  END IF;

  IF finance_fields_changed THEN
    NEW.classification_source := 'MANUAL';
    NEW.classification_policy_version := NULL;
    NEW.classification_status := CASE
      WHEN NEW.finance_category_level1_id IS NULL
        AND NEW.finance_category_level2_id IS NULL
      THEN 'UNCLASSIFIED'
      ELSE 'CLASSIFIED'
    END;
    IF TG_OP = 'INSERT'
      OR NEW.classification_updated_at IS NOT DISTINCT FROM OLD.classification_updated_at
    THEN
      NEW.classification_updated_at := clock_timestamp();
    END IF;
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER company_fund_finance_classification_ownership
BEFORE INSERT OR UPDATE
ON public.company_fund_transactions
FOR EACH ROW
EXECUTE FUNCTION public.company_fund_mark_manual_classification();

UPDATE public.company_fund_transactions
SET classification_source = 'MANUAL',
    classification_policy_version = NULL
WHERE finance_category_level1_id IS NOT NULL
   OR finance_category_level2_id IS NOT NULL
   OR is_operating_income_expense IS NOT NULL
   OR summary_inclusion_override IS NOT NULL
   OR classification_updated_by IS NOT NULL;
`

const migration062DownSQL = `
DROP TRIGGER IF EXISTS company_fund_finance_classification_ownership
  ON public.company_fund_transactions;
DROP FUNCTION IF EXISTS public.company_fund_mark_manual_classification();
DROP TABLE IF EXISTS public.company_fund_ledger_tasks;
DROP INDEX IF EXISTS public.idx_company_fund_transactions_classification_maintenance;
DROP INDEX IF EXISTS public.idx_company_fund_transactions_relationship_group;
DROP INDEX IF EXISTS public.idx_company_fund_transactions_relationship_reference;
ALTER TABLE public.company_fund_transactions
  DROP CONSTRAINT IF EXISTS company_fund_transactions_classification_policy_check,
  DROP CONSTRAINT IF EXISTS company_fund_transactions_classification_source_check,
  DROP CONSTRAINT IF EXISTS company_fund_transactions_relationship_reference_pair_check,
  DROP COLUMN IF EXISTS classification_policy_version,
  DROP COLUMN IF EXISTS classification_source,
  DROP COLUMN IF EXISTS relationship_group_key,
  DROP COLUMN IF EXISTS relationship_reference_key,
  DROP COLUMN IF EXISTS relationship_reference_type;
ALTER TABLE public.company_fund_provider_transaction_facts
  DROP COLUMN IF EXISTS provider_source_reference;
`
