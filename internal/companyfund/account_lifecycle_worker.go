package companyfund

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type AccountLifecycleCommandWorkerConfig struct {
	Owner         string
	LeaseDuration time.Duration
	RetryDelay    time.Duration
}

type AccountLifecycleCommandWorker struct {
	repository AccountLifecycleCommandRepository
	validator  AirwallexAccountIdentityValidator
	refresher  CompanyFundAccountRegistryRefresher
	config     AccountLifecycleCommandWorkerConfig
}

func NewAccountLifecycleCommandWorker(
	repository AccountLifecycleCommandRepository,
	validator AirwallexAccountIdentityValidator,
	refresher CompanyFundAccountRegistryRefresher,
	config AccountLifecycleCommandWorkerConfig,
) (*AccountLifecycleCommandWorker, error) {
	config.Owner = strings.TrimSpace(config.Owner)
	if repository == nil || refresher == nil || config.Owner == "" ||
		len(config.Owner) > 128 || config.LeaseDuration <= 0 || config.RetryDelay <= 0 {
		return nil, fmt.Errorf("invalid company-fund account lifecycle worker configuration")
	}
	return &AccountLifecycleCommandWorker{
		repository: repository,
		validator:  validator,
		refresher:  refresher,
		config:     config,
	}, nil
}

func (w *AccountLifecycleCommandWorker) ProcessNext(ctx context.Context) (AccountLifecycleProcessResult, error) {
	if ctx == nil {
		return AccountLifecycleProcessResult{}, fmt.Errorf("account lifecycle worker context is required")
	}
	lease, claimed, err := w.repository.ClaimAccountLifecycleCommand(
		ctx,
		w.config.Owner,
		w.config.LeaseDuration,
	)
	if err != nil {
		return AccountLifecycleProcessResult{}, fmt.Errorf("claim account lifecycle command: %w", err)
	}
	if !claimed {
		return AccountLifecycleProcessResult{Outcome: AccountLifecycleProcessIdle}, nil
	}
	result := AccountLifecycleProcessResult{
		Outcome:   AccountLifecycleProcessSucceeded,
		CommandID: lease.ID,
		Type:      lease.Type,
	}

	if !lease.BusinessApplied {
		identity, validationErr := w.validateProviderIdentity(ctx, lease)
		if validationErr != nil {
			if isRetryableAirwallexIdentityError(validationErr) {
				retry := AccountLifecycleFailure{
					CommandID:    lease.ID,
					LeaseOwner:   lease.LeaseOwner,
					AttemptCount: lease.AttemptCount,
					ErrorCode:    "PROVIDER_TEMPORARILY_UNAVAILABLE",
					SafeMessage:  "Airwallex is temporarily unavailable; validation will retry",
					RetryAfter:   w.config.RetryDelay,
				}
				if err := w.repository.RetryAccountLifecycleCommand(ctx, retry); err != nil {
					return result, fmt.Errorf("schedule provider validation retry: %w", err)
				}
				result.Outcome = AccountLifecycleProcessRetrying
				return result, nil
			}
			failure := AccountLifecycleFailure{
				CommandID:    lease.ID,
				LeaseOwner:   lease.LeaseOwner,
				AttemptCount: lease.AttemptCount,
				ErrorCode:    "PROVIDER_VALIDATION_FAILED",
				SafeMessage:  "Airwallex account identity validation failed",
			}
			if err := w.repository.FailAccountLifecycleCommand(ctx, failure); err != nil {
				return result, fmt.Errorf("finalize provider validation failure: %w", err)
			}
			result.Outcome = AccountLifecycleProcessFailed
			return result, nil
		}
		if err := w.repository.ApplyAccountLifecycleCommand(ctx, AccountLifecycleApplyInput{
			Lease:            lease,
			ProviderIdentity: identity,
		}); err != nil {
			errorCode := "STATE_CONFLICT"
			safeMessage := "Account lifecycle state or version no longer permits this command"
			if errors.Is(err, ErrAccountLifecycleUnfinishedProviderEvents) {
				errorCode = "PROVIDER_EVENTS_PENDING"
				safeMessage = "Prior Airwallex account still has provider events to process; retry cutover after they finish"
			}
			failure := AccountLifecycleFailure{
				CommandID:    lease.ID,
				LeaseOwner:   lease.LeaseOwner,
				AttemptCount: lease.AttemptCount,
				ErrorCode:    errorCode,
				SafeMessage:  safeMessage,
			}
			if finalizeErr := w.repository.FailAccountLifecycleCommand(ctx, failure); finalizeErr != nil {
				return result, fmt.Errorf("apply account lifecycle command: %v; finalize: %w", err, finalizeErr)
			}
			result.Outcome = AccountLifecycleProcessFailed
			return result, nil
		}
	}

	if err := w.refresher.Refresh(ctx); err != nil {
		failure := AccountLifecycleFailure{
			CommandID:       lease.ID,
			LeaseOwner:      lease.LeaseOwner,
			AttemptCount:    lease.AttemptCount,
			ErrorCode:       "REGISTRY_REFRESH_FAILED",
			SafeMessage:     "Account state changed but registry publication must retry",
			RetryAfter:      w.config.RetryDelay,
			BusinessApplied: true,
		}
		if retryErr := w.repository.RetryAccountLifecycleCommand(ctx, failure); retryErr != nil {
			return result, fmt.Errorf("retry registry publication: %w", retryErr)
		}
		result.Outcome = AccountLifecycleProcessRetrying
		return result, nil
	}

	if err := w.repository.CompleteAccountLifecycleCommand(
		ctx,
		lease.ID,
		lease.LeaseOwner,
		lease.AttemptCount,
	); err != nil {
		return result, fmt.Errorf("complete account lifecycle command: %w", err)
	}
	return result, nil
}

func isRetryableAirwallexIdentityError(err error) bool {
	return errors.Is(err, ErrAirwallexNetwork) ||
		errors.Is(err, ErrAirwallexServerResponse) ||
		errors.Is(err, ErrAirwallexResponseRead)
}

func (w *AccountLifecycleCommandWorker) validateProviderIdentity(
	ctx context.Context,
	lease AccountLifecycleCommandLease,
) (AirwallexProviderIdentitySummary, error) {
	switch lease.Type {
	case AccountLifecycleCommandPause, AccountLifecycleCommandDeleteCandidate:
		return AirwallexProviderIdentitySummary{}, nil
	case AccountLifecycleCommandValidateCandidate,
		AccountLifecycleCommandResume,
		AccountLifecycleCommandCutover:
		if w.validator == nil {
			return AirwallexProviderIdentitySummary{}, fmt.Errorf("Airwallex identity validator is unavailable")
		}
		return w.validator.ValidateAirwallexAccountIdentity(ctx, lease.TargetProviderKey)
	case AccountLifecycleCommandCorrectIdentity:
		if w.validator == nil {
			return AirwallexProviderIdentitySummary{}, fmt.Errorf("Airwallex identity validator is unavailable")
		}
		return w.validator.ValidateAirwallexAccountIdentity(ctx, lease.RequestedProviderKey)
	default:
		return AirwallexProviderIdentitySummary{}, fmt.Errorf("unsupported lifecycle command type")
	}
}

func (w *AccountLifecycleCommandWorker) NextAccountLifecycleCommandDue(
	ctx context.Context,
) (time.Time, error) {
	provider, ok := w.repository.(accountLifecycleCommandDueProvider)
	if !ok {
		return time.Time{}, nil
	}
	return provider.NextAccountLifecycleCommandDue(ctx)
}
