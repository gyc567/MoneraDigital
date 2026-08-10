package companyfund

import (
	"context"
	"errors"
	"time"
)

var ErrAccountLifecycleUnfinishedProviderEvents = errors.New(
	"Airwallex account has unfinished provider events",
)

type AirwallexAccountLifecycle string

const (
	AirwallexLifecycleCandidate AirwallexAccountLifecycle = "CANDIDATE"
	AirwallexLifecycleCurrent   AirwallexAccountLifecycle = "CURRENT"
	AirwallexLifecyclePaused    AirwallexAccountLifecycle = "PAUSED"
	AirwallexLifecycleRetired   AirwallexAccountLifecycle = "RETIRED"
	AirwallexLifecycleDeleted   AirwallexAccountLifecycle = "DELETED"
)

type AccountLifecycleCommandType string

const (
	AccountLifecycleCommandValidateCandidate AccountLifecycleCommandType = "VALIDATE_CANDIDATE"
	AccountLifecycleCommandPause             AccountLifecycleCommandType = "PAUSE"
	AccountLifecycleCommandResume            AccountLifecycleCommandType = "RESUME"
	AccountLifecycleCommandCorrectIdentity   AccountLifecycleCommandType = "CORRECT_IDENTITY"
	AccountLifecycleCommandCutover           AccountLifecycleCommandType = "CUTOVER"
	AccountLifecycleCommandDeleteCandidate   AccountLifecycleCommandType = "DELETE_CANDIDATE"
)

type AccountLifecycleCommandLease struct {
	ID                   int64
	Type                 AccountLifecycleCommandType
	TargetAccountID      int64
	RelatedAccountID     int64
	TargetProviderKey    string
	RequestedProviderKey string
	RequestedBy          string
	Reason               string
	ExpectedTargetVer    int64
	ExpectedRelatedVer   int64
	AttemptCount         int
	LeaseOwner           string
	LeaseExpiresAt       time.Time
	BusinessApplied      bool
}

// AirwallexProviderIdentitySummary contains only the minimum non-secret
// provider identity evidence safe to persist in the shared database.
type AirwallexProviderIdentitySummary struct {
	ProviderAccountID string `json:"providerAccountId"`
	AccountName       string `json:"accountName,omitempty"`
}

type AccountLifecycleApplyInput struct {
	Lease            AccountLifecycleCommandLease
	ProviderIdentity AirwallexProviderIdentitySummary
}

type AccountLifecycleFailure struct {
	CommandID       int64
	LeaseOwner      string
	AttemptCount    int
	ErrorCode       string
	SafeMessage     string
	RetryAfter      time.Duration
	BusinessApplied bool
}

type AccountLifecycleCommandRepository interface {
	ClaimAccountLifecycleCommand(context.Context, string, time.Duration) (AccountLifecycleCommandLease, bool, error)
	ApplyAccountLifecycleCommand(context.Context, AccountLifecycleApplyInput) error
	CompleteAccountLifecycleCommand(context.Context, int64, string, int) error
	FailAccountLifecycleCommand(context.Context, AccountLifecycleFailure) error
	RetryAccountLifecycleCommand(context.Context, AccountLifecycleFailure) error
}

type AccountLifecycleCommandProcessor interface {
	ProcessNext(context.Context) (AccountLifecycleProcessResult, error)
}

type accountLifecycleCommandDueProvider interface {
	NextAccountLifecycleCommandDue(context.Context) (time.Time, error)
}

type AirwallexAccountIdentityValidator interface {
	ValidateAirwallexAccountIdentity(context.Context, string) (AirwallexProviderIdentitySummary, error)
}

type CompanyFundAccountRegistryRefresher interface {
	Refresh(context.Context) error
}

type AccountLifecycleProcessOutcome string

const (
	AccountLifecycleProcessIdle      AccountLifecycleProcessOutcome = "IDLE"
	AccountLifecycleProcessSucceeded AccountLifecycleProcessOutcome = "SUCCEEDED"
	AccountLifecycleProcessFailed    AccountLifecycleProcessOutcome = "FAILED"
	AccountLifecycleProcessRetrying  AccountLifecycleProcessOutcome = "RETRYING"
)

type AccountLifecycleProcessResult struct {
	Outcome   AccountLifecycleProcessOutcome
	CommandID int64
	Type      AccountLifecycleCommandType
}
