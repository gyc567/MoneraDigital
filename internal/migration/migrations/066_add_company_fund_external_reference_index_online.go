package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"monera-digital/internal/migration"
)

// AddCompanyFundExternalReferenceIndexOnline creates the manual import lookup
// index without taking the transaction table offline for a normal index build.
type AddCompanyFundExternalReferenceIndexOnline struct{}

func (*AddCompanyFundExternalReferenceIndexOnline) Version() string { return "066" }

func (*AddCompanyFundExternalReferenceIndexOnline) Description() string {
	return "Create company-fund external reference index online"
}

func (*AddCompanyFundExternalReferenceIndexOnline) RequiredPreexistingVersion() string { return "065" }

func (*AddCompanyFundExternalReferenceIndexOnline) RequiredExpectedCeiling() string { return "066" }

func (*AddCompanyFundExternalReferenceIndexOnline) Up(*sql.DB) error {
	return fmt.Errorf("066 is online-controlled; run it through Migrator.MigrateWithExpectedCeiling")
}

func (*AddCompanyFundExternalReferenceIndexOnline) UpConn(ctx context.Context, conn *sql.Conn) error {
	if conn == nil {
		return fmt.Errorf("migration 066 dedicated session is required")
	}
	if _, err := conn.ExecContext(ctx, migration066TimeoutsSQL); err != nil {
		return fmt.Errorf("configure migration 066 timeouts: %w", err)
	}
	var invalid bool
	if err := conn.QueryRowContext(ctx, migration066InvalidIndexSQL).Scan(&invalid); err != nil {
		return fmt.Errorf("inspect migration 066 prior index state: %w", err)
	}
	if invalid {
		if _, err := conn.ExecContext(ctx, migration066DropInvalidIndexSQL); err != nil {
			return fmt.Errorf("remove invalid migration 066 index: %w", err)
		}
	}
	if _, err := conn.ExecContext(ctx, migration066CreateIndexSQL); err != nil {
		return fmt.Errorf("create migration 066 index concurrently: %w", err)
	}
	var valid bool
	if err := conn.QueryRowContext(ctx, migration066ValidateIndexSQL).Scan(&valid); err != nil {
		return fmt.Errorf("validate migration 066 index: %w", err)
	}
	if !valid {
		return fmt.Errorf("migration 066 index is not the expected valid and ready non-unique external reference index")
	}
	return nil
}

func (*AddCompanyFundExternalReferenceIndexOnline) Down(*sql.DB) error {
	return fmt.Errorf("066 is forward-only; external reference index changes require a new migration")
}

var _ migration.Migration = (*AddCompanyFundExternalReferenceIndexOnline)(nil)
var _ migration.ControlledOnlineMigration = (*AddCompanyFundExternalReferenceIndexOnline)(nil)

const migration066TimeoutsSQL = `SET search_path = pg_catalog, public; SET lock_timeout = '5s'; SET statement_timeout = '15min';`

const migration066InvalidIndexSQL = `
SELECT EXISTS (
  SELECT 1
  FROM pg_index AS idx
  JOIN pg_class AS index_class ON index_class.oid = idx.indexrelid
  JOIN pg_class AS table_class ON table_class.oid = idx.indrelid
  JOIN pg_namespace AS namespace ON namespace.oid = table_class.relnamespace
  WHERE namespace.nspname = 'public'
    AND table_class.relname = 'company_fund_transactions'
    AND index_class.relname = 'idx_company_fund_transactions_external_reference'
    AND (NOT idx.indisvalid OR NOT idx.indisready)
)`

const migration066DropInvalidIndexSQL = `DROP INDEX CONCURRENTLY IF EXISTS public.idx_company_fund_transactions_external_reference`

const migration066CreateIndexSQL = `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_company_fund_transactions_external_reference
  ON public.company_fund_transactions (external_transaction_reference)
  WHERE external_transaction_reference IS NOT NULL`

const migration066ValidateIndexSQL = `
SELECT EXISTS (
  SELECT 1
  FROM pg_index AS idx
  JOIN pg_class AS index_class ON index_class.oid = idx.indexrelid
  JOIN pg_class AS table_class ON table_class.oid = idx.indrelid
  JOIN pg_namespace AS namespace ON namespace.oid = table_class.relnamespace
  WHERE namespace.nspname = 'public'
    AND table_class.relname = 'company_fund_transactions'
    AND index_class.relname = 'idx_company_fund_transactions_external_reference'
    AND NOT idx.indisunique
    AND idx.indisvalid
    AND idx.indisready
    AND idx.indnkeyatts = 1
    AND (SELECT count(*)
         FROM unnest(idx.indkey) WITH ORDINALITY AS indexed_key(attnum, ordinal)
         JOIN pg_attribute AS attribute
           ON attribute.attrelid = idx.indrelid AND attribute.attnum = indexed_key.attnum
         WHERE indexed_key.ordinal <= idx.indnkeyatts
           AND attribute.attname = 'external_transaction_reference') = 1
    AND regexp_replace(lower(COALESCE(pg_get_expr(idx.indpred, idx.indrelid), '')), '[[:space:]()]', '', 'g')
      = 'external_transaction_referenceisnotnull'
)`
