package fundrouting

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresRoutingStatusCheckStoreSchedulesTxKeysAndCompletesClosedChecks(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err := NewPostgresRoutingStatusCheckStore(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("UPDATE safeheron_transaction_routing_status_checks status_check").
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec("INSERT INTO safeheron_transaction_routing_status_checks").
		WithArgs(int64((5 * time.Minute) / time.Millisecond)).
		WillReturnResult(sqlmock.NewResult(0, 2))

	scheduled, err := store.ScheduleOpen(context.Background(), 5*time.Minute)
	if err != nil || scheduled != 2 {
		t.Fatalf("ScheduleOpen() = %d, %v", scheduled, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRoutingStatusCheckStoreClaimsOneDueTxKeyWithLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, _ := NewPostgresRoutingStatusCheckStore(db)
	firstSeen := time.Date(2026, 8, 10, 3, 0, 0, 0, time.UTC)
	mock.ExpectQuery("WITH candidate AS").
		WithArgs("worker-1", int64(time.Minute/time.Millisecond)).
		WillReturnRows(sqlmock.NewRows([]string{"safeheron_tx_key", "first_seen_at", "attempt_count", "lease_owner"}).
			AddRow("tx-1", firstSeen, 2, "worker-1"))

	check, claimed, err := store.ClaimDue(context.Background(), "worker-1", time.Minute)
	if err != nil || !claimed || check.TxKey != "tx-1" || check.AttemptCount != 2 || !check.FirstSeenAt.Equal(firstSeen) || check.LeaseOwner != "worker-1" {
		t.Fatalf("ClaimDue() = %#v, %v, %v", check, claimed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRoutingStatusCheckStoreCompletesObservedOnlyForLeaseOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, _ := NewPostgresRoutingStatusCheckStore(db)
	checkedAt := time.Date(2026, 8, 10, 4, 0, 0, 0, time.UTC)
	next := checkedAt.Add(time.Hour)
	check := RoutingStatusCheck{TxKey: "tx-1", LeaseOwner: "worker-1"}
	eventID := "safeheron-routing-status:v1:event-1"
	mock.ExpectExec("UPDATE safeheron_transaction_routing_status_checks").
		WithArgs("tx-1", "worker-1", checkedAt, "SUBMITTED", false, &next, eventID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.CompleteObserved(context.Background(), routingStatusObserved{
		Check: check, CheckedAt: checkedAt, Status: "SUBMITTED", EventID: eventID,
		Terminal: false, NextCheckAt: &next,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRoutingStatusCheckStoreCompletesFailureWithBoundedCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, _ := NewPostgresRoutingStatusCheckStore(db)
	checkedAt := time.Date(2026, 8, 10, 4, 0, 0, 0, time.UTC)
	next := checkedAt.Add(15 * time.Minute)
	mock.ExpectExec("UPDATE safeheron_transaction_routing_status_checks").
		WithArgs("tx-1", "worker-1", checkedAt, "PROVIDER_LOOKUP_FAILED", next).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.CompleteFailed(context.Background(), routingStatusFailure{
		Check: RoutingStatusCheck{TxKey: "tx-1", LeaseOwner: "worker-1"}, CheckedAt: checkedAt,
		ErrorCode: "PROVIDER_LOOKUP_FAILED", NextCheckAt: next,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRoutingStatusCheckStoreRejectsLostLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, _ := NewPostgresRoutingStatusCheckStore(db)
	checkedAt := time.Date(2026, 8, 10, 4, 0, 0, 0, time.UTC)
	next := checkedAt.Add(15 * time.Minute)
	mock.ExpectExec("UPDATE safeheron_transaction_routing_status_checks").
		WithArgs("tx-1", "another-worker", checkedAt, "SUBMITTED", false, &next, "safeheron-routing-status:v1:event-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.CompleteObserved(context.Background(), routingStatusObserved{
		Check: RoutingStatusCheck{TxKey: "tx-1", LeaseOwner: "another-worker"}, CheckedAt: checkedAt,
		Status: "SUBMITTED", EventID: "safeheron-routing-status:v1:event-1", NextCheckAt: &next,
	})
	if err == nil || !strings.Contains(err.Error(), "lease was lost") {
		t.Fatalf("CompleteObserved() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRoutingStatusCheckStoreNextDueIncludesUnscheduledOpenCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, _ := NewPostgresRoutingStatusCheckStore(db)
	due := time.Date(2026, 8, 10, 3, 5, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT min\\(due_at\\)").
		WithArgs(int64((5 * time.Minute) / time.Millisecond)).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(due))

	got, err := store.NextDue(context.Background(), 5*time.Minute)
	if err != nil || !got.Equal(due) {
		t.Fatalf("NextDue() = %s, %v", got, err)
	}
}
