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
