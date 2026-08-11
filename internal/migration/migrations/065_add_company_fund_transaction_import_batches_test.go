package migrations

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"monera-digital/internal/migration"
)

func TestMigration065DefinesControlledManualImportBatchContract(t *testing.T) {
	value := &AddCompanyFundTransactionImportBatches{}
	var controlled migration.ControlledMigration = value
	if controlled.Version() != "065" || controlled.RequiredPreexistingVersion() != "064" || controlled.RequiredExpectedCeiling() != "065" {
		t.Fatalf("controlled migration contract = %s/%s/%s", controlled.Version(), controlled.RequiredPreexistingVersion(), controlled.RequiredExpectedCeiling())
	}
	if value.Description() == "" {
		t.Fatal("migration description must not be empty")
	}
	if err := value.Up(nil); err == nil || !strings.Contains(err.Error(), "controlled") {
		t.Fatalf("direct Up() = %v", err)
	}
	if err := value.Down(nil); err == nil || !strings.Contains(err.Error(), "forward-only") {
		t.Fatalf("Down() = %v", err)
	}
	if err := value.UpTx(nil); err == nil || !strings.Contains(err.Error(), "transaction is required") {
		t.Fatalf("nil UpTx() = %v", err)
	}
}

func TestMigration065RunsOnlyInsideControlledTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migration065TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(migration065SchemaSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := (&AddCompanyFundTransactionImportBatches{}).UpTx(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration065ReturnsSchemaFailureForRollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migration065TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(migration065SchemaSQL)).WillReturnError(errors.New("ddl failed"))
	mock.ExpectRollback()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := (&AddCompanyFundTransactionImportBatches{}).UpTx(tx); err == nil {
		t.Fatal("expected migration 065 DDL failure")
	}
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration065ReturnsTimeoutFailureForRollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migration065TimeoutsSQL)).WillReturnError(errors.New("timeout failed"))
	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := (&AddCompanyFundTransactionImportBatches{}).UpTx(tx); err == nil || !strings.Contains(err.Error(), "configure migration 065 timeouts") {
		t.Fatalf("UpTx() = %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration065CreatesAuditableImportBatchesWithoutProviderIdentityReuse(t *testing.T) {
	for _, fragment := range []string{
		"ADD COLUMN external_transaction_reference VARCHAR(256)",
		"CREATE TABLE public.company_fund_transaction_import_batches",
		"CREATE TABLE public.company_fund_transaction_import_rows",
		"content_digest VARCHAR(64) NOT NULL",
		"idempotency_key VARCHAR(128) NOT NULL",
		"duplicate_warning_evidence JSONB NOT NULL DEFAULT '[]'::jsonb",
		"started_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()",
		"jsonb_typeof(duplicate_warning_evidence) = 'array'",
		"UNIQUE (requested_by, idempotency_key)",
		"WHERE status IN ('PROCESSING', 'SUCCEEDED')",
		"principal_transaction_id BIGINT NOT NULL",
		"fee_transaction_id BIGINT",
		"voided_movement_count INTEGER NOT NULL DEFAULT 0",
		"voided_movement_count <= principal_transaction_count + fee_transaction_count",
		"finance_category_level1_id BIGINT",
		"finance_category_level2_id BIGINT",
		"UNIQUE (batch_id, row_digest)",
		"UNIQUE (principal_transaction_id)",
		"UNIQUE (fee_transaction_id)",
		"company_fund_transaction_import_rows_movement_ownership_exclude",
		"int8multirange(",
		"WITH &&",
		"predecessor_batch_id BIGINT",
		"reimport_reason TEXT",
		"voided_at TIMESTAMPTZ",
		"company_fund_validate_import_batch_lineage",
		"company_fund_enforce_import_row_transaction_ownership",
		"company_fund_validate_import_batch_counts",
		"company_fund_validate_import_batch_counts_trigger",
		"AFTER INSERT OR UPDATE OR DELETE",
		"terminal import batch status is immutable",
		"ORDER BY transaction_row.id",
		"FOR UPDATE",
		"import principal transaction must be a MANUAL ADJUSTMENT",
		"import fee transaction must be a MANUAL FEE OUTFLOW",
		"imported transaction account must match the import row account",
		"a voided import batch can have only one effective replacement",
	} {
		if !strings.Contains(migration065SchemaSQL, fragment) {
			t.Errorf("migration 065 schema is missing %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"provider_transaction_id",
		"provider_reported_fee_amount",
		"UNIQUE (external_transaction_reference)",
		"DROP TABLE",
	} {
		if strings.Contains(migration065SchemaSQL, forbidden) {
			t.Errorf("migration 065 must not repurpose provider identity or fee facts: %q", forbidden)
		}
	}
	for _, match := range regexp.MustCompile(`(?m)(?:CONSTRAINT|INDEX|TRIGGER|FUNCTION)\s+(?:public\.)?([a-z_][a-z0-9_]*)`).FindAllStringSubmatch(migration065SchemaSQL, -1) {
		if len(match[1]) > 63 {
			t.Errorf("migration 065 identifier exceeds PostgreSQL's 63-byte identifier limit: %s", match[1])
		}
	}
}
