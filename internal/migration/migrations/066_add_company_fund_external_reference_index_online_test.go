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

func TestMigration066DefinesOnlineControlledIndexContract(t *testing.T) {
	value := &AddCompanyFundExternalReferenceIndexOnline{}
	var controlled migration.ControlledOnlineMigration = value
	if controlled.Version() != "066" || controlled.RequiredPreexistingVersion() != "065" || controlled.RequiredExpectedCeiling() != "066" {
		t.Fatalf("online controlled migration contract = %s/%s/%s", controlled.Version(), controlled.RequiredPreexistingVersion(), controlled.RequiredExpectedCeiling())
	}
	if err := value.Up(nil); err == nil || !strings.Contains(err.Error(), "online-controlled") {
		t.Fatalf("direct Up() = %v", err)
	}
	if err := value.UpConn(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "dedicated session") {
		t.Fatalf("nil UpConn() = %v", err)
	}
	if err := value.Down(nil); err == nil || !strings.Contains(err.Error(), "forward-only") {
		t.Fatalf("Down() = %v", err)
	}
	if !strings.Contains(migration066CreateIndexSQL, "btrim(external_transaction_reference)") {
		t.Fatalf("migration 066 must index the normalized external reference: %s", migration066CreateIndexSQL)
	}
	if !strings.Contains(migration066ValidateIndexSQL, "btrim(external_transaction_reference::text)") {
		t.Fatalf("migration 066 validation must require the normalized expression: %s", migration066ValidateIndexSQL)
	}
	for _, fragment := range []string{"idx.indnatts = 1", "access_method.amname = 'btree'"} {
		if !strings.Contains(migration066ValidateIndexSQL, fragment) {
			t.Fatalf("migration 066 validation must reject index shape drift: %s", fragment)
		}
	}
}

func TestMigration066CreatesAndValidatesExternalReferenceIndexOutsideTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	mock.ExpectExec(regexp.QuoteMeta(migration066TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(migration066InvalidIndexSQL)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(regexp.QuoteMeta(migration066CreateIndexSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(migration066ValidateIndexSQL)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	if err := (&AddCompanyFundExternalReferenceIndexOnline{}).UpConn(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration066RebuildsOnlyAnInvalidPriorIndexAndFailsClosedOnInvalidResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	mock.ExpectExec(regexp.QuoteMeta(migration066TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(migration066InvalidIndexSQL)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta(migration066DropInvalidIndexSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(migration066CreateIndexSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(migration066ValidateIndexSQL)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err = (&AddCompanyFundExternalReferenceIndexOnline{}).UpConn(context.Background(), conn)
	if err == nil || !strings.Contains(err.Error(), "not the expected valid and ready") {
		t.Fatalf("UpConn() = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration066WrapsSessionFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	mock.ExpectExec(regexp.QuoteMeta(migration066TimeoutsSQL)).WillReturnError(errors.New("timeout failed"))

	err = (&AddCompanyFundExternalReferenceIndexOnline{}).UpConn(context.Background(), conn)
	if err == nil || !strings.Contains(err.Error(), "configure migration 066 timeouts") {
		t.Fatalf("UpConn() = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigration066WrapsEveryCatalogAndDDLFailure(t *testing.T) {
	testCases := []struct {
		name        string
		prepare     func(sqlmock.Sqlmock)
		wantMessage string
	}{
		{
			name: "prior index inspection",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(migration066TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta(migration066InvalidIndexSQL)).WillReturnError(errors.New("catalog unavailable"))
			},
			wantMessage: "inspect migration 066 prior index state",
		},
		{
			name: "invalid index removal",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(migration066TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta(migration066InvalidIndexSQL)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectExec(regexp.QuoteMeta(migration066DropInvalidIndexSQL)).WillReturnError(errors.New("drop failed"))
			},
			wantMessage: "remove invalid migration 066 index",
		},
		{
			name: "concurrent index creation",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(migration066TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta(migration066InvalidIndexSQL)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
				mock.ExpectExec(regexp.QuoteMeta(migration066CreateIndexSQL)).WillReturnError(errors.New("create failed"))
			},
			wantMessage: "create migration 066 index concurrently",
		},
		{
			name: "postcondition inspection",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(migration066TimeoutsSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta(migration066InvalidIndexSQL)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
				mock.ExpectExec(regexp.QuoteMeta(migration066CreateIndexSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta(migration066ValidateIndexSQL)).WillReturnError(errors.New("validate failed"))
			},
			wantMessage: "validate migration 066 index",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			conn, err := db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Close() })
			testCase.prepare(mock)

			err = (&AddCompanyFundExternalReferenceIndexOnline{}).UpConn(context.Background(), conn)
			if err == nil || !strings.Contains(err.Error(), testCase.wantMessage) {
				t.Fatalf("UpConn() = %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
