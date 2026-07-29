package migrations

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"monera-digital/internal/migration"
)

func TestMigration063MetadataIsControlledAndForwardOnly(t *testing.T) {
	value := &AddAirwallexAccountLifecycle{}
	var controlled migration.ControlledMigration = value
	if controlled.Version() != "063" || value.Description() == "" {
		t.Fatal("unexpected migration 063 metadata")
	}
	if value.RequiredPreexistingVersion() != "062" || value.RequiredExpectedCeiling() != "063" {
		t.Fatal("migration 063 must require the immediately preceding controlled ceiling")
	}
	if err := value.Up(nil); err == nil || !strings.Contains(err.Error(), "controlled") {
		t.Fatalf("direct Up() error = %v", err)
	}
	if err := value.Down(nil); err == nil || !strings.Contains(err.Error(), "forward-only") {
		t.Fatalf("Down() error = %v", err)
	}
}

func TestMigration063AddsLifecycleAndDurableCommandContract(t *testing.T) {
	sqlText := migration063SchemaSQL
	for _, required := range []string{
		"ADD COLUMN airwallex_lifecycle VARCHAR(16)",
		"ADD COLUMN lifecycle_version BIGINT NOT NULL DEFAULT 1",
		"ADD COLUMN airwallex_validated_at TIMESTAMPTZ",
		"ADD COLUMN airwallex_provider_identity_summary JSONB",
		"ADD COLUMN deleted_at TIMESTAMPTZ",
		"ADD COLUMN deleted_by VARCHAR(255)",
		"ADD COLUMN delete_reason VARCHAR(1000)",
		"'CANDIDATE', 'CURRENT', 'PAUSED', 'RETIRED', 'DELETED'",
		"CREATE UNIQUE INDEX uq_company_fund_accounts_airwallex_current",
		"CREATE UNIQUE INDEX uq_company_fund_accounts_airwallex_candidate",
		"CREATE TABLE public.company_fund_account_lifecycle_commands",
		"business_applied_at TIMESTAMPTZ",
		"CREATE TABLE public.company_fund_account_lifecycle_audits",
		"old_provider_account_key VARCHAR(128)",
		"new_provider_account_key VARCHAR(128)",
		"CREATE TABLE public.company_fund_classification_policy_bindings",
		"finance_category_level1_id BIGINT NOT NULL",
		"finance_category_level2_id BIGINT NOT NULL",
		"CREATE TRIGGER trg_finance_categories_system_binding_guard",
		"CREATE TRIGGER trg_company_fund_classification_binding_hierarchy_guard",
		"CREATE TRIGGER trg_company_fund_provider_events_airwallex_scope_guard",
		"CREATE TRIGGER trg_company_fund_provider_facts_airwallex_scope_guard",
		"CREATE TRIGGER trg_company_fund_transactions_airwallex_scope_guard",
		"CREATE TRIGGER trg_company_fund_ledger_tasks_airwallex_scope_guard",
		"CREATE TRIGGER trg_company_fund_account_asset_policies_airwallex_scope_guard",
		"pg_advisory_xact_lock_shared",
		"system classification policy binding",
		"'VALIDATE_CANDIDATE'",
		"'PAUSE'",
		"'RESUME'",
		"'CORRECT_IDENTITY'",
		"'CUTOVER'",
		"'DELETE_CANDIDATE'",
		"'PENDING', 'PROCESSING', 'SUCCEEDED', 'FAILED'",
		"idempotency_key VARCHAR(255) NOT NULL",
		"expected_target_version BIGINT NOT NULL",
		"lease_owner VARCHAR(128)",
		"lease_expires_at TIMESTAMPTZ",
		"next_attempt_at TIMESTAMPTZ NOT NULL",
		"CREATE UNIQUE INDEX uq_company_fund_account_lifecycle_commands_inflight",
		"CREATE INDEX idx_company_fund_account_lifecycle_commands_due",
		"CREATE TRIGGER trg_company_fund_accounts_airwallex_command_guard",
	} {
		if !strings.Contains(sqlText, required) {
			t.Errorf("migration 063 must include %q", required)
		}
	}
	for _, forbidden := range []string{
		"api_key",
		"bearer",
		"access_token",
		"webhook_secret",
		"provider_payload",
	} {
		if strings.Contains(strings.ToLower(sqlText), forbidden) {
			t.Errorf("migration 063 must not persist sensitive field %q", forbidden)
		}
	}
}

func TestMigration063RejectsAmbiguousLegacyAirwallexAccounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migration063TimeoutsSQL)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(migration063LegacyPreflightSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"violations", "account_ids", "account_details"}).
			AddRow(2, "4,7", `[{"id":4,"isEnabled":true,"wasEnabled":true},{"id":7,"isEnabled":false,"wasEnabled":true}]`))
	mock.ExpectRollback()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	err = (&AddAirwallexAccountLifecycle{}).UpTx(tx)
	if err == nil || !strings.Contains(err.Error(), "violations=2") {
		t.Fatalf("UpTx() error = %v, want legacy preflight rejection", err)
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration063AcceptsExactExplicitLegacyLifecycleMapping(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migration063TimeoutsSQL)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(migration063LegacyPreflightSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"violations", "account_ids", "account_details"}).
			AddRow(2, "4,7", `[{"id":4,"isEnabled":true,"wasEnabled":true},{"id":7,"isEnabled":false,"wasEnabled":true}]`))
	mock.ExpectExec(regexp.QuoteMeta(migration063CreateLegacyMappingTableSQL)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(migration063InsertLegacyMappingSQL)).
		WithArgs(int64(4), "CURRENT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(migration063InsertLegacyMappingSQL)).
		WithArgs(int64(7), "RETIRED").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(migration063SchemaSQL)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	migration := &AddAirwallexAccountLifecycle{
		LegacyMappingJSON: `{"4":"CURRENT","7":"RETIRED"}`,
	}
	if err := migration.UpTx(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration063ExecutesInsideControlledRunnerTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migration063TimeoutsSQL)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(migration063LegacyPreflightSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"violations", "account_ids", "account_details"}).AddRow(0, "", `[]`))
	mock.ExpectExec(regexp.QuoteMeta(migration063CreateLegacyMappingTableSQL)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(migration063SchemaSQL)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := (&AddAirwallexAccountLifecycle{}).UpTx(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration063ReturnsDDLFailuresForRollback(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  string
	}{
		{
			name: "timeouts",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(migration063TimeoutsSQL)).
					WillReturnError(errors.New("timeout"))
			},
			want: "timeouts",
		},
		{
			name: "preflight",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(migration063TimeoutsSQL)).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta(migration063LegacyPreflightSQL)).
					WillReturnError(errors.New("query"))
			},
			want: "preflight",
		},
		{
			name: "schema",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(migration063TimeoutsSQL)).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta(migration063LegacyPreflightSQL)).
					WillReturnRows(sqlmock.NewRows([]string{"violations", "account_ids", "account_details"}).AddRow(0, "", `[]`))
				mock.ExpectExec(regexp.QuoteMeta(migration063CreateLegacyMappingTableSQL)).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(regexp.QuoteMeta(migration063SchemaSQL)).
					WillReturnError(errors.New("ddl"))
			},
			want: "lifecycle",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			mock.ExpectBegin()
			testCase.setup(mock)
			mock.ExpectRollback()
			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			err = (&AddAirwallexAccountLifecycle{}).UpTx(tx)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("UpTx() error = %v, want %q", err, testCase.want)
			}
			_ = tx.Rollback()
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
