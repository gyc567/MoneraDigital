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

func TestMigration062MetadataIsControlledAndReversible(t *testing.T) {
	value := &AddAirwallexPhase2Ledger{}
	var controlled migration.ControlledMigration = value
	if controlled.Version() != "062" || value.Description() == "" {
		t.Fatal("unexpected migration 062 metadata")
	}
	if value.RequiredPreexistingVersion() != "061" || value.RequiredExpectedCeiling() != "062" {
		t.Fatal("migration 062 must require the immediately preceding controlled ceiling")
	}
	if err := value.Up(nil); err == nil || !strings.Contains(err.Error(), "controlled") {
		t.Fatal("migration 062 must reject uncontrolled Up")
	}
}

func TestMigration062AddsDurableRelationshipAndClassificationContract(t *testing.T) {
	sqlText := migration062SchemaSQL
	for _, required := range []string{
		"ADD COLUMN provider_source_reference VARCHAR(256)",
		"ADD COLUMN relationship_reference_type VARCHAR(32)",
		"ADD COLUMN relationship_reference_key VARCHAR(256)",
		"ADD COLUMN relationship_group_key VARCHAR(256)",
		"ADD COLUMN classification_source VARCHAR(32)",
		"ADD COLUMN classification_policy_version VARCHAR(64)",
		"SOURCE_ID_GROUP_ONLY",
		"CREATE TABLE public.company_fund_ledger_tasks",
		"provider_transaction_fact_id BIGINT NOT NULL",
		"task_payload TEXT NOT NULL",
		"attempt_count INTEGER NOT NULL DEFAULT 0",
		"next_attempt_at TIMESTAMPTZ NOT NULL",
		"lease_owner VARCHAR(128)",
		"lease_expires_at TIMESTAMPTZ",
		"sla_expires_at TIMESTAMPTZ NOT NULL",
		"terminal_at TIMESTAMPTZ",
		"last_error_code VARCHAR(128)",
		"CREATE UNIQUE INDEX uq_company_fund_ledger_tasks_identity",
		"COALESCE(policy_version, '')",
		"CREATE TRIGGER company_fund_finance_classification_ownership",
		"NEW.classification_source := 'MANUAL'",
	} {
		if !strings.Contains(sqlText, required) {
			t.Fatalf("migration 062 must include %q", required)
		}
	}
	for _, forbidden := range []string{
		"provider_payload",
		"owned_payload",
		"bearer",
		"api_key",
		"webhook_secret",
	} {
		if strings.Contains(strings.ToLower(sqlText), forbidden) {
			t.Fatalf("migration 062 must not persist sensitive field %q", forbidden)
		}
	}
}

func TestMigration062CannotBypassManualOwnershipByWritingSource(t *testing.T) {
	if strings.Contains(migration062SchemaSQL,
		"NEW.classification_source IS NOT DISTINCT FROM OLD.classification_source",
	) {
		t.Fatal("direct finance-field writes must become MANUAL even when the caller also writes classification_source")
	}
	if !strings.Contains(migration062SchemaSQL,
		"current_setting('monera.company_fund_classification_origin', true) = 'SYSTEM'",
	) {
		t.Fatal("system automation must use the transaction-local origin guard")
	}
}

func TestMigration062ExecutesInsideControlledRunnerTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migration062TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(migration062SchemaSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := (&AddAirwallexPhase2Ledger{}).UpTx(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration062DownRemovesOnlyPhase2Objects(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta(migration062DownSQL)).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := (&AddAirwallexPhase2Ledger{}).Down(db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"DROP TABLE public.company_fund_transactions",
		"DROP TABLE public.company_fund_provider_transaction_facts",
		"DELETE FROM",
	} {
		if strings.Contains(migration062DownSQL, forbidden) {
			t.Fatalf("migration 062 down contains destructive unrelated SQL %q", forbidden)
		}
	}
}

func TestMigration062ReturnsDDLFailureForRollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migration062TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(migration062SchemaSQL)).WillReturnError(errors.New("ddl failed"))
	mock.ExpectRollback()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := (&AddAirwallexPhase2Ledger{}).UpTx(tx); err == nil {
		t.Fatal("expected migration 062 DDL failure")
	}
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		t.Fatal(err)
	}
}
