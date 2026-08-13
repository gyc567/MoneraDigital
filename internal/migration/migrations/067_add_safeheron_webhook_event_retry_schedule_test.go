package migrations

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"monera-digital/internal/migration"
)

func TestAddSafeheronWebhookEventRetryScheduleContract(t *testing.T) {
	var controlled migration.ControlledMigration = &AddSafeheronWebhookEventRetrySchedule{}
	if controlled.Version() != "067" || controlled.RequiredPreexistingVersion() != "066" || controlled.RequiredExpectedCeiling() != "067" {
		t.Fatalf("controlled migration contract = %s/%s/%s", controlled.Version(), controlled.RequiredPreexistingVersion(), controlled.RequiredExpectedCeiling())
	}
	if controlled.Description() == "" {
		t.Fatal("migration description must not be empty")
	}
}

func TestMigration067RequiresControlledTransactionAndIsForwardOnly(t *testing.T) {
	migration := &AddSafeheronWebhookEventRetrySchedule{}
	if err := migration.Up(nil); err == nil || !strings.Contains(err.Error(), "controlled") {
		t.Fatalf("Up() = %v", err)
	}
	if err := migration.UpTx(nil); err == nil || !strings.Contains(err.Error(), "transaction is required") {
		t.Fatalf("UpTx(nil) = %v", err)
	}
	if err := migration.Down(nil); err == nil || !strings.Contains(err.Error(), "forward-only") {
		t.Fatalf("Down() = %v", err)
	}
}

func TestMigration067WrapsTransactionalFailures(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		prepare     func(sqlmock.Sqlmock)
		wantMessage string
	}{
		{
			name: "timeouts",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(migration067TimeoutsSQL)).WillReturnError(errors.New("timeout setup failed"))
			},
			wantMessage: "configure migration 067 timeouts",
		},
		{
			name: "schema",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(migration067TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(regexp.QuoteMeta(migration067SchemaSQL)).WillReturnError(errors.New("schema failed"))
			},
			wantMessage: "install Safeheron webhook event retry schedule",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := db.Close(); err != nil {
					t.Errorf("close migration 067 test database: %v", err)
				}
			})
			mock.ExpectBegin()
			tx, err := db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			testCase.prepare(mock)

			err = (&AddSafeheronWebhookEventRetrySchedule{}).UpTx(tx)
			if err == nil || !strings.Contains(err.Error(), testCase.wantMessage) {
				t.Fatalf("UpTx() = %v, want %q", err, testCase.wantMessage)
			}
			mock.ExpectRollback()
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			mock.ExpectClose()
		})
	}
}

func TestMigration067AddsBoundedPendingEventRetrySchedule(t *testing.T) {
	for _, fragment := range []string{
		"ADD COLUMN next_attempt_at TIMESTAMPTZ",
		"process_status = 'PENDING' OR next_attempt_at IS NULL",
		"idx_safeheron_webhook_events_pending_retry",
		"WHERE process_status = 'PENDING'",
	} {
		if !strings.Contains(migration067SchemaSQL, fragment) {
			t.Errorf("migration 067 schema is missing %q", fragment)
		}
	}
}
