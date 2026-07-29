package companyfund

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeAccountLifecycleRepository struct {
	lease       AccountLifecycleCommandLease
	claimed     bool
	claimErr    error
	applied     []AccountLifecycleApplyInput
	applyErr    error
	completed   int
	failed      []AccountLifecycleFailure
	retried     []AccountLifecycleFailure
	completeErr error
	finalizeErr error
}

func (f *fakeAccountLifecycleRepository) ClaimAccountLifecycleCommand(context.Context, string, time.Duration) (AccountLifecycleCommandLease, bool, error) {
	return f.lease, f.claimed, f.claimErr
}

func (f *fakeAccountLifecycleRepository) ApplyAccountLifecycleCommand(_ context.Context, input AccountLifecycleApplyInput) error {
	f.applied = append(f.applied, input)
	return f.applyErr
}

func (f *fakeAccountLifecycleRepository) CompleteAccountLifecycleCommand(context.Context, int64, string, int) error {
	f.completed++
	return f.completeErr
}

func (f *fakeAccountLifecycleRepository) FailAccountLifecycleCommand(_ context.Context, failure AccountLifecycleFailure) error {
	f.failed = append(f.failed, failure)
	return f.finalizeErr
}

func (f *fakeAccountLifecycleRepository) RetryAccountLifecycleCommand(_ context.Context, failure AccountLifecycleFailure) error {
	f.retried = append(f.retried, failure)
	return f.finalizeErr
}

type fakeAirwallexAccountIdentityValidator struct {
	keys    []string
	summary AirwallexProviderIdentitySummary
	err     error
}

func (f *fakeAirwallexAccountIdentityValidator) ValidateAirwallexAccountIdentity(_ context.Context, key string) (AirwallexProviderIdentitySummary, error) {
	f.keys = append(f.keys, key)
	return f.summary, f.err
}

type fakeAccountRegistryRefresher struct {
	calls int
	err   error
}

func (f *fakeAccountRegistryRefresher) Refresh(context.Context) error {
	f.calls++
	return f.err
}

func newAccountLifecycleWorkerForTest(
	t *testing.T,
	repository AccountLifecycleCommandRepository,
	validator AirwallexAccountIdentityValidator,
	refresher CompanyFundAccountRegistryRefresher,
) *AccountLifecycleCommandWorker {
	t.Helper()
	worker, err := NewAccountLifecycleCommandWorker(
		repository,
		validator,
		refresher,
		AccountLifecycleCommandWorkerConfig{
			Owner:         "test-worker",
			LeaseDuration: time.Minute,
			RetryDelay:    time.Second,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return worker
}

func TestAccountLifecycleWorker_ValidationCommandsUseExactProviderIdentity(t *testing.T) {
	for _, command := range []AccountLifecycleCommandType{
		AccountLifecycleCommandValidateCandidate,
		AccountLifecycleCommandResume,
		AccountLifecycleCommandCutover,
	} {
		t.Run(string(command), func(t *testing.T) {
			repository := &fakeAccountLifecycleRepository{
				claimed: true,
				lease: AccountLifecycleCommandLease{
					ID:                11,
					Type:              command,
					TargetAccountID:   7,
					TargetProviderKey: "AcCt-Case-Sensitive",
					ExpectedTargetVer: 3,
					AttemptCount:      1,
					LeaseOwner:        "test-worker",
					RequestedBy:       "finance@example.com",
					Reason:            "approved",
				},
			}
			validator := &fakeAirwallexAccountIdentityValidator{
				summary: AirwallexProviderIdentitySummary{ProviderAccountID: "AcCt-Case-Sensitive"},
			}
			refresher := &fakeAccountRegistryRefresher{}
			result, err := newAccountLifecycleWorkerForTest(t, repository, validator, refresher).
				ProcessNext(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if result.Outcome != AccountLifecycleProcessSucceeded ||
				len(validator.keys) != 1 || validator.keys[0] != "AcCt-Case-Sensitive" ||
				len(repository.applied) != 1 || refresher.calls != 1 || repository.completed != 1 {
				t.Fatalf("unexpected process result: %#v validator=%v repo=%#v refresh=%d",
					result, validator.keys, repository, refresher.calls)
			}
		})
	}
}

func TestAccountLifecycleWorker_CorrectionValidatesRequestedIdentity(t *testing.T) {
	repository := &fakeAccountLifecycleRepository{
		claimed: true,
		lease: AccountLifecycleCommandLease{
			ID:                   12,
			Type:                 AccountLifecycleCommandCorrectIdentity,
			TargetAccountID:      7,
			TargetProviderKey:    "old-id",
			RequestedProviderKey: "new-id",
			ExpectedTargetVer:    4,
			AttemptCount:         1,
			LeaseOwner:           "test-worker",
			RequestedBy:          "finance@example.com",
			Reason:               "fix typo",
		},
	}
	validator := &fakeAirwallexAccountIdentityValidator{
		summary: AirwallexProviderIdentitySummary{ProviderAccountID: "new-id"},
	}
	result, err := newAccountLifecycleWorkerForTest(
		t, repository, validator, &fakeAccountRegistryRefresher{},
	).ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != AccountLifecycleProcessSucceeded ||
		len(validator.keys) != 1 || validator.keys[0] != "new-id" ||
		repository.applied[0].ProviderIdentity.ProviderAccountID != "new-id" {
		t.Fatalf("unexpected correction result: %#v keys=%v apply=%#v",
			result, validator.keys, repository.applied)
	}
}

func TestAccountLifecycleWorker_PauseAndDeleteRemainProviderIndependent(t *testing.T) {
	for _, command := range []AccountLifecycleCommandType{
		AccountLifecycleCommandPause,
		AccountLifecycleCommandDeleteCandidate,
	} {
		t.Run(string(command), func(t *testing.T) {
			repository := &fakeAccountLifecycleRepository{
				claimed: true,
				lease: AccountLifecycleCommandLease{
					ID:                13,
					Type:              command,
					TargetAccountID:   7,
					TargetProviderKey: "provider-down",
					ExpectedTargetVer: 2,
					AttemptCount:      1,
					LeaseOwner:        "test-worker",
					RequestedBy:       "finance@example.com",
					Reason:            "emergency",
				},
			}
			validator := &fakeAirwallexAccountIdentityValidator{err: errors.New("provider unavailable")}
			result, err := newAccountLifecycleWorkerForTest(
				t, repository, validator, &fakeAccountRegistryRefresher{},
			).ProcessNext(context.Background())
			if err != nil || result.Outcome != AccountLifecycleProcessSucceeded ||
				len(validator.keys) != 0 {
				t.Fatalf("provider-independent command result=%#v err=%v keys=%v",
					result, err, validator.keys)
			}
		})
	}
}

func TestAccountLifecycleWorker_ProviderFailureIsSafeTerminalFailure(t *testing.T) {
	repository := &fakeAccountLifecycleRepository{
		claimed: true,
		lease: AccountLifecycleCommandLease{
			ID: 14, Type: AccountLifecycleCommandValidateCandidate,
			TargetProviderKey: "bad-id", ExpectedTargetVer: 1,
			AttemptCount: 1, LeaseOwner: "test-worker",
		},
	}
	validator := &fakeAirwallexAccountIdentityValidator{err: ErrAirwallexUnauthorized}
	result, err := newAccountLifecycleWorkerForTest(
		t, repository, validator, &fakeAccountRegistryRefresher{},
	).ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != AccountLifecycleProcessFailed || len(repository.applied) != 0 ||
		len(repository.failed) != 1 || repository.failed[0].ErrorCode != "PROVIDER_VALIDATION_FAILED" ||
		repository.failed[0].SafeMessage == "" {
		t.Fatalf("unexpected provider failure: %#v repo=%#v", result, repository)
	}
}

func TestAccountLifecycleWorker_TransientProviderFailureUsesDurableRetry(t *testing.T) {
	for _, transient := range []error{
		ErrAirwallexNetwork,
		ErrAirwallexServerResponse,
		ErrAirwallexResponseRead,
	} {
		repository := &fakeAccountLifecycleRepository{
			claimed: true,
			lease: AccountLifecycleCommandLease{
				ID: 140, Type: AccountLifecycleCommandValidateCandidate,
				TargetProviderKey: "candidate-id", ExpectedTargetVer: 1,
				AttemptCount: 2, LeaseOwner: "test-worker",
			},
		}
		validator := &fakeAirwallexAccountIdentityValidator{err: transient}
		result, err := newAccountLifecycleWorkerForTest(
			t, repository, validator, &fakeAccountRegistryRefresher{},
		).ProcessNext(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if result.Outcome != AccountLifecycleProcessRetrying ||
			len(repository.retried) != 1 || len(repository.failed) != 0 ||
			repository.retried[0].ErrorCode != "PROVIDER_TEMPORARILY_UNAVAILABLE" ||
			repository.retried[0].RetryAfter != time.Second {
			t.Fatalf("transient=%v result=%#v repo=%#v", transient, result, repository)
		}
	}
}

func TestAccountLifecycleWorker_RegistryRefreshFailureRetriesOnlyPublication(t *testing.T) {
	repository := &fakeAccountLifecycleRepository{
		claimed: true,
		lease: AccountLifecycleCommandLease{
			ID: 15, Type: AccountLifecycleCommandPause,
			TargetProviderKey: "account-id", ExpectedTargetVer: 2,
			AttemptCount: 1, LeaseOwner: "test-worker",
		},
	}
	refresher := &fakeAccountRegistryRefresher{err: errors.New("database unavailable")}
	result, err := newAccountLifecycleWorkerForTest(
		t, repository, &fakeAirwallexAccountIdentityValidator{}, refresher,
	).ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != AccountLifecycleProcessRetrying ||
		len(repository.applied) != 1 || len(repository.retried) != 1 ||
		repository.completed != 0 || !repository.retried[0].BusinessApplied {
		t.Fatalf("unexpected refresh retry: %#v repo=%#v", result, repository)
	}

	repository.lease.BusinessApplied = true
	repository.retried = nil
	refresher.err = nil
	result, err = newAccountLifecycleWorkerForTest(
		t, repository, &fakeAirwallexAccountIdentityValidator{}, refresher,
	).ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != AccountLifecycleProcessSucceeded ||
		len(repository.applied) != 1 || repository.completed != 1 {
		t.Fatalf("recovery repeated business mutation: result=%#v repo=%#v", result, repository)
	}
}

func TestAccountLifecycleWorker_CutoverQueueConflictIsActionable(t *testing.T) {
	repository := &fakeAccountLifecycleRepository{
		claimed: true,
		lease: AccountLifecycleCommandLease{
			ID: 16, Type: AccountLifecycleCommandCutover,
			TargetProviderKey: "candidate-id", ExpectedTargetVer: 2,
			AttemptCount: 1, LeaseOwner: "test-worker",
		},
		applyErr: ErrAccountLifecycleUnfinishedProviderEvents,
	}
	validator := &fakeAirwallexAccountIdentityValidator{
		summary: AirwallexProviderIdentitySummary{ProviderAccountID: "candidate-id"},
	}
	result, err := newAccountLifecycleWorkerForTest(
		t, repository, validator, &fakeAccountRegistryRefresher{},
	).ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != AccountLifecycleProcessFailed ||
		len(repository.failed) != 1 ||
		repository.failed[0].ErrorCode != "PROVIDER_EVENTS_PENDING" ||
		!strings.Contains(repository.failed[0].SafeMessage, "retry cutover") {
		t.Fatalf("unexpected cutover queue failure: result=%#v repo=%#v", result, repository)
	}
}

func TestAccountLifecycleWorker_IdleAndInfrastructureErrors(t *testing.T) {
	t.Run("idle", func(t *testing.T) {
		result, err := newAccountLifecycleWorkerForTest(
			t,
			&fakeAccountLifecycleRepository{},
			&fakeAirwallexAccountIdentityValidator{},
			&fakeAccountRegistryRefresher{},
		).ProcessNext(context.Background())
		if err != nil || result.Outcome != AccountLifecycleProcessIdle {
			t.Fatalf("result=%#v err=%v", result, err)
		}
	})
	t.Run("claim error", func(t *testing.T) {
		_, err := newAccountLifecycleWorkerForTest(
			t,
			&fakeAccountLifecycleRepository{claimErr: errors.New("db")},
			&fakeAirwallexAccountIdentityValidator{},
			&fakeAccountRegistryRefresher{},
		).ProcessNext(context.Background())
		if err == nil {
			t.Fatal("expected claim error")
		}
	})
}
