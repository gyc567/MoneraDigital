package companyfund

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAccountLifecycleRepository_ClaimRecoversExpiredLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	expires := time.Now().UTC().Add(time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta(claimAccountLifecycleCommandSQL)).
		WithArgs("worker-1", int64(time.Minute/time.Microsecond)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "command_type", "target_account_id", "related_account_id",
			"target_provider_account_key", "requested_provider_account_key",
			"requested_by", "reason", "expected_target_version",
			"expected_related_version", "attempt_count", "lease_owner",
			"lease_expires_at", "business_applied",
		}).AddRow(
			int64(41), "CUTOVER", int64(9), int64(8), "candidate-id", nil,
			"finance@example.com", "rotate", int64(2), int64(3), 4,
			"worker-1", expires, false,
		))

	lease, claimed, err := NewDBRepository(db).ClaimAccountLifecycleCommand(
		context.Background(), "worker-1", time.Minute,
	)
	if err != nil || !claimed || lease.ID != 41 ||
		lease.Type != AccountLifecycleCommandCutover || lease.RelatedAccountID != 8 ||
		lease.ExpectedRelatedVer != 3 || lease.AttemptCount != 4 {
		t.Fatalf("lease=%#v claimed=%t err=%v", lease, claimed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountLifecycleRepository_ClaimIdle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery(regexp.QuoteMeta(claimAccountLifecycleCommandSQL)).
		WithArgs("worker-1", int64(time.Minute/time.Microsecond)).
		WillReturnError(sql.ErrNoRows)
	_, claimed, err := NewDBRepository(db).ClaimAccountLifecycleCommand(
		context.Background(), "worker-1", time.Minute,
	)
	if err != nil || claimed {
		t.Fatalf("claimed=%t err=%v", claimed, err)
	}
}

func TestAccountLifecycleRepository_RetryPreservesBusinessApplicationPhase(t *testing.T) {
	if strings.Contains(completeAccountLifecycleCommandSQL, "$7") {
		t.Fatal("completion SQL must not depend on the retry phase parameter")
	}
	if !strings.Contains(retryAccountLifecycleCommandSQL, "$7::boolean") {
		t.Fatal("retry SQL must bind and enforce the requested business application phase")
	}
	for _, businessApplied := range []bool{false, true} {
		t.Run(map[bool]string{false: "before business apply", true: "after business apply"}[businessApplied], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			mock.ExpectExec(regexp.QuoteMeta(retryAccountLifecycleCommandSQL)).
				WithArgs(
					int64(42),
					"worker-1",
					3,
					int64(time.Second/time.Microsecond),
					"PROVIDER_TEMPORARILY_UNAVAILABLE",
					"retry safely",
					businessApplied,
				).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err = NewDBRepository(db).RetryAccountLifecycleCommand(
				context.Background(),
				AccountLifecycleFailure{
					CommandID:       42,
					LeaseOwner:      "worker-1",
					AttemptCount:    3,
					ErrorCode:       "PROVIDER_TEMPORARILY_UNAVAILABLE",
					SafeMessage:     "retry safely",
					RetryAfter:      time.Second,
					BusinessApplied: businessApplied,
				},
			)
			if err != nil {
				t.Fatalf("RetryAccountLifecycleCommand() error = %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAccountLifecycleTransitionPolicy(t *testing.T) {
	for _, testCase := range []struct {
		command AccountLifecycleCommandType
		from    AirwallexAccountLifecycle
		valid   bool
	}{
		{AccountLifecycleCommandValidateCandidate, AirwallexLifecycleCandidate, true},
		{AccountLifecycleCommandPause, AirwallexLifecycleCurrent, true},
		{AccountLifecycleCommandResume, AirwallexLifecyclePaused, true},
		{AccountLifecycleCommandCorrectIdentity, AirwallexLifecycleCandidate, true},
		{AccountLifecycleCommandCorrectIdentity, AirwallexLifecycleCurrent, true},
		{AccountLifecycleCommandCorrectIdentity, AirwallexLifecyclePaused, true},
		{AccountLifecycleCommandCutover, AirwallexLifecycleCandidate, true},
		{AccountLifecycleCommandDeleteCandidate, AirwallexLifecycleCandidate, true},
		{AccountLifecycleCommandResume, AirwallexLifecycleRetired, false},
		{AccountLifecycleCommandCorrectIdentity, AirwallexLifecycleRetired, false},
		{AccountLifecycleCommandDeleteCandidate, AirwallexLifecycleDeleted, false},
	} {
		err := validateAccountLifecycleCommandSource(testCase.command, testCase.from)
		if (err == nil) != testCase.valid {
			t.Errorf("command=%s from=%s err=%v valid=%t", testCase.command, testCase.from, err, testCase.valid)
		}
	}
}

func TestAccountLifecycleCorrectionAndDeletionReferenceManifestsAreComplete(t *testing.T) {
	for _, table := range []string{
		"company_fund_provider_events",
		"company_fund_provider_transaction_facts",
		"company_fund_transactions",
		"company_fund_ledger_tasks",
	} {
		if !strings.Contains(correctAirwallexProviderAccountReferencesSQL, table) {
			t.Errorf("correction manifest omits %s", table)
		}
	}
	for _, reference := range []string{
		"company_fund_account_asset_policies",
		"company_fund_provider_events",
		"company_fund_provider_transaction_facts",
		"company_fund_transactions",
		"company_fund_ledger_tasks",
		"status IN ('PENDING', 'PROCESSING')",
	} {
		if !strings.Contains(countAirwallexCandidateReferencesSQL, reference) {
			t.Errorf("deletion reference check omits %s", reference)
		}
	}
}
