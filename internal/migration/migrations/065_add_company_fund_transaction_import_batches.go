package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"monera-digital/internal/migration"
)

// AddCompanyFundTransactionImportBatches adds the durable shared contract used
// by the management application to atomically import manual ledger movements.
// It deliberately keeps external references separate from Provider identities.
type AddCompanyFundTransactionImportBatches struct{}

func (*AddCompanyFundTransactionImportBatches) Version() string { return "065" }

func (*AddCompanyFundTransactionImportBatches) Description() string {
	return "Add durable company-fund manual transaction import batches"
}

func (*AddCompanyFundTransactionImportBatches) RequiredPreexistingVersion() string { return "064" }

func (*AddCompanyFundTransactionImportBatches) RequiredExpectedCeiling() string { return "065" }

func (*AddCompanyFundTransactionImportBatches) Up(*sql.DB) error {
	return fmt.Errorf("065 is controlled; run it through Migrator.MigrateWithExpectedCeiling")
}

func (*AddCompanyFundTransactionImportBatches) UpTx(tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("migration 065 transaction is required")
	}
	ctx := context.Background()
	if _, err := tx.ExecContext(ctx, migration065TimeoutsSQL); err != nil {
		return fmt.Errorf("configure migration 065 timeouts: %w", err)
	}
	if _, err := tx.ExecContext(ctx, migration065SchemaSQL); err != nil {
		return fmt.Errorf("install company-fund transaction import batch schema: %w", err)
	}
	return nil
}

func (*AddCompanyFundTransactionImportBatches) Down(*sql.DB) error {
	return fmt.Errorf("065 is forward-only; transaction import batches must be changed by a new migration")
}

var _ migration.Migration = (*AddCompanyFundTransactionImportBatches)(nil)
var _ migration.ControlledMigration = (*AddCompanyFundTransactionImportBatches)(nil)

const migration065TimeoutsSQL = `SET LOCAL search_path = pg_catalog, public; SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '60s'; SET LOCAL idle_in_transaction_session_timeout = '60s';`

const migration065SchemaSQL = `
ALTER TABLE public.company_fund_transactions
  ADD COLUMN external_transaction_reference VARCHAR(256);

ALTER TABLE public.company_fund_transactions
  ADD CONSTRAINT company_fund_transactions_external_reference_source_check
  CHECK (
    external_transaction_reference IS NULL
    OR (
      channel = 'MANUAL'
      AND btrim(external_transaction_reference) <> ''
    )
  );

CREATE TABLE public.company_fund_transaction_import_batches (
  id BIGSERIAL PRIMARY KEY,
  content_digest VARCHAR(64) NOT NULL
    CONSTRAINT company_fund_transaction_import_batches_content_digest_check
    CHECK (content_digest ~ '^[0-9a-f]{64}$'),
  request_digest VARCHAR(64) NOT NULL
    CONSTRAINT company_fund_transaction_import_batches_request_digest_check
    CHECK (request_digest ~ '^[0-9a-f]{64}$'),
  template_version VARCHAR(32) NOT NULL
    CONSTRAINT company_fund_transaction_import_batches_template_version_check
    CHECK (btrim(template_version) <> ''),
  original_file_name VARCHAR(255) NOT NULL
    CONSTRAINT company_fund_import_batches_original_file_name_check
    CHECK (btrim(original_file_name) <> ''),
  status VARCHAR(16) NOT NULL
    CONSTRAINT company_fund_transaction_import_batches_status_check
    CHECK (status IN ('PROCESSING', 'SUCCEEDED', 'FAILED', 'VOIDED')),
  requested_by BIGINT NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL
    CONSTRAINT company_fund_transaction_import_batches_idempotency_key_check
    CHECK (btrim(idempotency_key) <> ''),
  source_row_count INTEGER NOT NULL
    CONSTRAINT company_fund_transaction_import_batches_source_row_count_check
    CHECK (source_row_count BETWEEN 1 AND 500),
  principal_transaction_count INTEGER NOT NULL DEFAULT 0
    CONSTRAINT company_fund_import_batches_principal_tx_count_check
    CHECK (principal_transaction_count >= 0),
  fee_transaction_count INTEGER NOT NULL DEFAULT 0
    CONSTRAINT company_fund_import_batches_fee_tx_count_check
    CHECK (fee_transaction_count >= 0),
  voided_movement_count INTEGER NOT NULL DEFAULT 0
    CONSTRAINT company_fund_import_batches_voided_count_nonnegative_check
    CHECK (voided_movement_count >= 0),
  warning_count INTEGER NOT NULL DEFAULT 0
    CONSTRAINT company_fund_transaction_import_batches_warning_count_check
    CHECK (warning_count >= 0),
  duplicate_override_acknowledged BOOLEAN NOT NULL DEFAULT false,
  duplicate_override_reason TEXT,
  duplicate_warning_evidence JSONB NOT NULL DEFAULT '[]'::jsonb
    CONSTRAINT company_fund_import_batches_duplicate_warning_evidence_check
    CHECK (jsonb_typeof(duplicate_warning_evidence) = 'array'),
  predecessor_batch_id BIGINT
    CONSTRAINT company_fund_transaction_import_batches_predecessor_fk
    REFERENCES public.company_fund_transaction_import_batches(id) ON DELETE RESTRICT,
  reimport_reason TEXT,
  failure_code VARCHAR(64),
  failure_summary VARCHAR(512),
  started_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  completed_at TIMESTAMPTZ,
  voided_at TIMESTAMPTZ,
  voided_by BIGINT,
  void_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  CONSTRAINT company_fund_transaction_import_batches_idempotency_unique
    UNIQUE (requested_by, idempotency_key),
  CONSTRAINT company_fund_transaction_import_batches_predecessor_self_check
    CHECK (predecessor_batch_id IS NULL OR predecessor_batch_id <> id),
  CONSTRAINT company_fund_import_batches_voided_count_check
    CHECK (voided_movement_count <= principal_transaction_count + fee_transaction_count),
  CONSTRAINT company_fund_transaction_import_batches_reimport_check CHECK (
    (predecessor_batch_id IS NULL AND reimport_reason IS NULL)
    OR (
      predecessor_batch_id IS NOT NULL
      AND reimport_reason IS NOT NULL
      AND btrim(reimport_reason) <> ''
    )
  ),
  CONSTRAINT company_fund_import_batches_duplicate_override_check CHECK (
    (duplicate_override_acknowledged
      AND duplicate_override_reason IS NOT NULL
      AND btrim(duplicate_override_reason) <> '')
    OR (
      NOT duplicate_override_acknowledged
      AND duplicate_override_reason IS NULL
    )
  ),
  CONSTRAINT company_fund_transaction_import_batches_lifecycle_check CHECK (
    (status = 'PROCESSING'
      AND completed_at IS NULL
      AND failure_code IS NULL
      AND failure_summary IS NULL
      AND voided_movement_count = 0
      AND voided_at IS NULL
      AND voided_by IS NULL
      AND void_reason IS NULL)
    OR (status = 'SUCCEEDED'
      AND completed_at IS NOT NULL
      AND failure_code IS NULL
      AND failure_summary IS NULL
      AND voided_movement_count = 0
      AND voided_at IS NULL
      AND voided_by IS NULL
      AND void_reason IS NULL)
    OR (status = 'FAILED'
      AND completed_at IS NOT NULL
      AND failure_code IS NOT NULL
      AND failure_summary IS NOT NULL
      AND voided_movement_count = 0
      AND voided_at IS NULL
      AND voided_by IS NULL
      AND void_reason IS NULL)
    OR (status = 'VOIDED'
      AND completed_at IS NOT NULL
      AND failure_code IS NULL
      AND failure_summary IS NULL
      AND voided_at IS NOT NULL
      AND voided_by IS NOT NULL
      AND void_reason IS NOT NULL
      AND btrim(void_reason) <> '')
  )
);

CREATE UNIQUE INDEX uq_company_fund_transaction_import_batches_effective_content
  ON public.company_fund_transaction_import_batches (content_digest)
  WHERE status IN ('PROCESSING', 'SUCCEEDED');

CREATE INDEX idx_company_fund_transaction_import_batches_created
  ON public.company_fund_transaction_import_batches (created_at DESC, id DESC);

CREATE INDEX idx_company_fund_transaction_import_batches_predecessor
  ON public.company_fund_transaction_import_batches (predecessor_batch_id)
  WHERE predecessor_batch_id IS NOT NULL;

CREATE OR REPLACE FUNCTION public.company_fund_validate_import_batch_lineage()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  predecessor public.company_fund_transaction_import_batches%ROWTYPE;
BEGIN
  IF NEW.predecessor_batch_id IS NOT NULL THEN
    SELECT * INTO predecessor
    FROM public.company_fund_transaction_import_batches
    WHERE id = NEW.predecessor_batch_id
    FOR UPDATE;
    IF NOT FOUND OR predecessor.status <> 'VOIDED' THEN
      RAISE EXCEPTION 'import predecessor must be a voided batch';
    END IF;
    IF predecessor.content_digest <> NEW.content_digest THEN
      RAISE EXCEPTION 'import predecessor must have the same content digest';
    END IF;
    IF EXISTS (
      SELECT 1
      FROM public.company_fund_transaction_import_batches AS sibling
      WHERE sibling.predecessor_batch_id = NEW.predecessor_batch_id
        AND sibling.id <> NEW.id
        AND sibling.status <> 'FAILED'
    ) THEN
      RAISE EXCEPTION 'a voided import batch can have only one effective replacement';
    END IF;
    IF EXISTS (
      WITH RECURSIVE lineage(id) AS (
        SELECT NEW.predecessor_batch_id
        UNION ALL
        SELECT batch.predecessor_batch_id
        FROM public.company_fund_transaction_import_batches AS batch
        JOIN lineage ON lineage.id = batch.id
        WHERE batch.predecessor_batch_id IS NOT NULL
      )
      SELECT 1 FROM lineage WHERE id = NEW.id
    ) THEN
      RAISE EXCEPTION 'import predecessor chain cannot contain a cycle';
    END IF;
  ELSIF NEW.status <> 'FAILED' AND EXISTS (
    SELECT 1
    FROM public.company_fund_transaction_import_batches AS sibling
    WHERE sibling.content_digest = NEW.content_digest
      AND sibling.id <> NEW.id
      AND sibling.status <> 'FAILED'
  ) THEN
    RAISE EXCEPTION 'an existing import source requires an explicit voided predecessor';
  END IF;

  IF NEW.status <> 'VOIDED' AND EXISTS (
    SELECT 1
    FROM public.company_fund_transaction_import_batches AS child
    WHERE child.predecessor_batch_id = NEW.id
      AND child.status <> 'FAILED'
  ) THEN
    RAISE EXCEPTION 'a batch with an effective replacement must remain voided';
  END IF;
  RETURN NEW;
END;
$$;

CREATE CONSTRAINT TRIGGER company_fund_validate_import_batch_lineage_trigger
AFTER INSERT OR UPDATE OF status, content_digest, predecessor_batch_id
ON public.company_fund_transaction_import_batches
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW
EXECUTE FUNCTION public.company_fund_validate_import_batch_lineage();

CREATE TABLE public.company_fund_transaction_import_rows (
  id BIGSERIAL PRIMARY KEY,
  batch_id BIGINT NOT NULL
    CONSTRAINT company_fund_transaction_import_rows_batch_fk
    REFERENCES public.company_fund_transaction_import_batches(id) ON DELETE RESTRICT,
  source_row_number INTEGER NOT NULL CONSTRAINT company_fund_transaction_import_rows_source_row_number_check CHECK (source_row_number >= 2),
  row_digest VARCHAR(64) NOT NULL CONSTRAINT company_fund_transaction_import_rows_row_digest_check CHECK (row_digest ~ '^[0-9a-f]{64}$'),
  external_transaction_reference VARCHAR(256),
  company_fund_account_id BIGINT NOT NULL
    CONSTRAINT company_fund_transaction_import_rows_account_fk
    REFERENCES public.company_fund_accounts(id) ON DELETE RESTRICT,
  finance_category_level1_id BIGINT
    CONSTRAINT company_fund_transaction_import_rows_category_level1_fk
    REFERENCES public.finance_categories(id) ON DELETE RESTRICT,
  finance_category_level2_id BIGINT
    CONSTRAINT company_fund_transaction_import_rows_category_level2_fk
    REFERENCES public.finance_categories(id) ON DELETE RESTRICT,
  principal_transaction_id BIGINT NOT NULL
    CONSTRAINT company_fund_transaction_import_rows_principal_fk
    REFERENCES public.company_fund_transactions(id) ON DELETE RESTRICT,
  fee_transaction_id BIGINT
    CONSTRAINT company_fund_transaction_import_rows_fee_fk
    REFERENCES public.company_fund_transactions(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  CONSTRAINT company_fund_transaction_import_rows_source_row_unique UNIQUE (batch_id, source_row_number),
  CONSTRAINT company_fund_transaction_import_rows_row_digest_unique UNIQUE (batch_id, row_digest),
  CONSTRAINT company_fund_transaction_import_rows_principal_unique UNIQUE (principal_transaction_id),
  CONSTRAINT company_fund_transaction_import_rows_fee_unique UNIQUE (fee_transaction_id),
  CONSTRAINT company_fund_transaction_import_rows_movement_ownership_exclude
    EXCLUDE USING gist (
      int8multirange(
        int8range(principal_transaction_id, principal_transaction_id, '[]'),
        CASE WHEN fee_transaction_id IS NULL
          THEN 'empty'::int8range
          ELSE int8range(fee_transaction_id, fee_transaction_id, '[]')
        END
      ) WITH &&
    ),
  CONSTRAINT company_fund_import_rows_principal_fee_distinct_check CHECK (fee_transaction_id IS NULL OR fee_transaction_id <> principal_transaction_id),
  CONSTRAINT company_fund_transaction_import_rows_category_hierarchy_check CHECK (finance_category_level2_id IS NULL OR finance_category_level1_id IS NOT NULL)
);

CREATE OR REPLACE FUNCTION public.company_fund_enforce_import_row_transaction_ownership()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  import_batch public.company_fund_transaction_import_batches%ROWTYPE;
  principal_tx public.company_fund_transactions%ROWTYPE;
  fee_tx public.company_fund_transactions%ROWTYPE;
  target_batch_id BIGINT;
BEGIN
  IF TG_OP = 'UPDATE' AND NEW.batch_id IS DISTINCT FROM OLD.batch_id THEN
    RAISE EXCEPTION 'an import row cannot move between batches';
  END IF;
  target_batch_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.batch_id ELSE NEW.batch_id END;
  SELECT * INTO import_batch
  FROM public.company_fund_transaction_import_batches
  WHERE id = target_batch_id
  FOR UPDATE;
  IF NOT FOUND OR import_batch.status <> 'PROCESSING' THEN
    RAISE EXCEPTION 'import rows can be changed only while the batch is processing';
  END IF;
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;

  PERFORM 1
  FROM public.company_fund_transactions AS transaction_row
  WHERE transaction_row.id IN (NEW.principal_transaction_id, NEW.fee_transaction_id)
  ORDER BY transaction_row.id
  FOR UPDATE;

  SELECT * INTO principal_tx
  FROM public.company_fund_transactions
  WHERE id = NEW.principal_transaction_id;
  IF NOT FOUND
    OR principal_tx.channel <> 'MANUAL'
    OR principal_tx.movement_kind <> 'ADJUSTMENT'
    OR principal_tx.transaction_direction NOT IN ('INFLOW', 'OUTFLOW') THEN
    RAISE EXCEPTION 'import principal transaction must be a MANUAL ADJUSTMENT';
  END IF;
  IF (principal_tx.transaction_direction = 'INFLOW'
      AND principal_tx.to_company_fund_account_id IS DISTINCT FROM NEW.company_fund_account_id)
    OR (principal_tx.transaction_direction = 'OUTFLOW'
      AND principal_tx.from_company_fund_account_id IS DISTINCT FROM NEW.company_fund_account_id) THEN
    RAISE EXCEPTION 'imported transaction account must match the import row account';
  END IF;
  IF principal_tx.external_transaction_reference
      IS DISTINCT FROM NEW.external_transaction_reference THEN
    RAISE EXCEPTION 'import principal transaction reference must match the import row';
  END IF;
  IF NEW.fee_transaction_id IS NOT NULL THEN
    SELECT * INTO fee_tx
    FROM public.company_fund_transactions
    WHERE id = NEW.fee_transaction_id;
    IF NOT FOUND
      OR fee_tx.channel <> 'MANUAL'
      OR fee_tx.movement_kind <> 'FEE'
      OR fee_tx.transaction_direction <> 'OUTFLOW' THEN
      RAISE EXCEPTION 'import fee transaction must be a MANUAL FEE OUTFLOW';
    END IF;
    IF fee_tx.from_company_fund_account_id IS DISTINCT FROM NEW.company_fund_account_id THEN
      RAISE EXCEPTION 'imported transaction account must match the import row account';
    END IF;
    IF fee_tx.external_transaction_reference
        IS DISTINCT FROM NEW.external_transaction_reference THEN
      RAISE EXCEPTION 'import fee transaction reference must match the import row';
    END IF;
  END IF;
  IF EXISTS (
    SELECT 1
    FROM public.company_fund_transaction_import_rows AS existing
    WHERE existing.id <> NEW.id
      AND (
        NEW.principal_transaction_id = existing.principal_transaction_id
        OR NEW.principal_transaction_id = existing.fee_transaction_id
        OR (NEW.fee_transaction_id IS NOT NULL AND NEW.fee_transaction_id = existing.principal_transaction_id)
        OR (NEW.fee_transaction_id IS NOT NULL AND NEW.fee_transaction_id = existing.fee_transaction_id)
      )
  ) THEN
    RAISE EXCEPTION 'an imported movement can belong to only one import row';
  END IF;
  RETURN NEW;
END;
$$;

CREATE CONSTRAINT TRIGGER company_fund_enforce_import_row_transaction_ownership_trigger
AFTER INSERT OR UPDATE OR DELETE
ON public.company_fund_transaction_import_rows
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW
EXECUTE FUNCTION public.company_fund_enforce_import_row_transaction_ownership();

CREATE OR REPLACE FUNCTION public.company_fund_validate_import_batch_counts()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  actual_source_rows INTEGER;
  actual_principal_transactions INTEGER;
  actual_fee_transactions INTEGER;
BEGIN
  IF TG_OP = 'INSERT' AND NEW.status <> 'PROCESSING' THEN
    RAISE EXCEPTION 'an import batch must start in processing';
  END IF;
  IF TG_OP = 'UPDATE' AND NEW.status IS DISTINCT FROM OLD.status
    AND NOT (
      (OLD.status = 'PROCESSING' AND NEW.status IN ('SUCCEEDED', 'FAILED'))
      OR (OLD.status = 'SUCCEEDED' AND NEW.status = 'VOIDED')
    ) THEN
    RAISE EXCEPTION 'invalid import batch status transition';
  END IF;

  SELECT count(*)::integer, count(principal_transaction_id)::integer,
    count(fee_transaction_id)::integer
  INTO actual_source_rows, actual_principal_transactions, actual_fee_transactions
  FROM public.company_fund_transaction_import_rows
  WHERE batch_id = NEW.id;

  IF NEW.status = 'FAILED' THEN
    IF actual_source_rows <> 0
      OR NEW.principal_transaction_count <> 0
      OR NEW.fee_transaction_count <> 0 THEN
      RAISE EXCEPTION 'failed import batches cannot retain durable movement rows';
    END IF;
    RETURN NEW;
  END IF;
  IF NEW.status NOT IN ('SUCCEEDED', 'VOIDED') THEN
    RETURN NEW;
  END IF;

  IF actual_source_rows = 0
    OR actual_source_rows <> NEW.source_row_count
    OR actual_principal_transactions <> NEW.principal_transaction_count
    OR actual_fee_transactions <> NEW.fee_transaction_count THEN
    RAISE EXCEPTION 'terminal import batch counts must match its durable import rows';
  END IF;
  RETURN NEW;
END;
$$;

CREATE CONSTRAINT TRIGGER company_fund_validate_import_batch_counts_trigger
AFTER INSERT OR UPDATE OF status, source_row_count,
  principal_transaction_count, fee_transaction_count
ON public.company_fund_transaction_import_batches
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW
EXECUTE FUNCTION public.company_fund_validate_import_batch_counts();
`
