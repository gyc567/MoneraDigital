package fundrouting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"monera-digital/internal/adaptiveschedule"
	"monera-digital/internal/companyfund"
	"monera-digital/internal/safeheron"
)

const (
	defaultRoutingStatusInitialDelay = 5 * time.Minute
	defaultRoutingStatusLease        = time.Minute
)

var defaultRoutingStatusSchedule = []time.Duration{
	5 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
	time.Hour,
}

// RoutingStatusCheck is one leased txKey-level provider lookup. A batch
// transaction can own several routing cases, but it must still consume only
// one Safeheron /v1/transactions/one request per schedule point.
type RoutingStatusCheck struct {
	TxKey        string
	FirstSeenAt  time.Time
	AttemptCount int
	LeaseOwner   string
}

type routingStatusObserved struct {
	Check       RoutingStatusCheck
	CheckedAt   time.Time
	Status      string
	EventID     string
	Terminal    bool
	NextCheckAt *time.Time
}

type routingStatusFailure struct {
	Check       RoutingStatusCheck
	CheckedAt   time.Time
	ErrorCode   string
	NextCheckAt time.Time
}

type RoutingStatusCheckStore interface {
	ScheduleOpen(context.Context, time.Duration) (int64, error)
	ClaimDue(context.Context, string, time.Duration) (RoutingStatusCheck, bool, error)
	CompleteObserved(context.Context, routingStatusObserved) error
	CompleteFailed(context.Context, routingStatusFailure) error
	NextDue(context.Context, time.Duration) (time.Time, error)
}

type RoutingStatusLookup interface {
	LookupTransaction(context.Context, safeheron.TransactionLookup) (*safeheron.TransactionSnapshot, error)
}

type RoutingStatusSnapshotIngester interface {
	Ingest(context.Context, companyfund.OwnedProviderPayloadInput) (companyfund.ProviderEventInsertResult, error)
}

type StatusRefresherConfig struct {
	WorkerID     string
	Lease        time.Duration
	InitialDelay time.Duration
	Now          func() time.Time
}

type StatusRefresher struct {
	store            RoutingStatusCheckStore
	lookup           RoutingStatusLookup
	ingester         RoutingStatusSnapshotIngester
	config           StatusRefresherConfig
	runner           *adaptiveRunner
	onSnapshotStored func()
	onCheckCompleted func()
}

func NewStatusRefresher(
	store RoutingStatusCheckStore,
	lookup RoutingStatusLookup,
	ingester RoutingStatusSnapshotIngester,
	config StatusRefresherConfig,
) (*StatusRefresher, error) {
	if store == nil || lookup == nil || ingester == nil {
		return nil, fmt.Errorf("Safeheron routing status store, lookup client, and ingester are required")
	}
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = newProjectionWorkerID()
	}
	if config.Lease <= 0 {
		config.Lease = defaultRoutingStatusLease
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = defaultRoutingStatusInitialDelay
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	refresher := &StatusRefresher{store: store, lookup: lookup, ingester: ingester, config: config}
	refresher.runner = newAdaptiveRunner(
		"Safeheron routing non-terminal status refresher",
		time.Minute,
		adaptiveschedule.DefaultMaxIdle,
		refresher.ProcessOne,
	)
	refresher.runner.setNextDue(refresher.NextDue)
	return refresher, nil
}

func (r *StatusRefresher) SetOnSnapshotStored(fn func()) {
	if r != nil {
		r.onSnapshotStored = fn
	}
}

func (r *StatusRefresher) SetOnCheckCompleted(fn func()) {
	if r != nil {
		r.onCheckCompleted = fn
	}
}

func (r *StatusRefresher) Notify() bool {
	if r == nil || r.runner == nil {
		return false
	}
	return r.runner.Notify()
}

func (r *StatusRefresher) NextDue(ctx context.Context) (time.Time, error) {
	if r == nil || r.store == nil {
		return time.Time{}, nil
	}
	return r.store.NextDue(ctx, r.config.InitialDelay)
}

func (r *StatusRefresher) ProcessOne(ctx context.Context) (bool, error) {
	if r == nil || r.store == nil || r.lookup == nil || r.ingester == nil {
		return false, fmt.Errorf("Safeheron routing status refresher is not configured")
	}
	scheduled, err := r.store.ScheduleOpen(ctx, r.config.InitialDelay)
	if err != nil {
		return false, fmt.Errorf("schedule Safeheron routing status checks: %w", err)
	}
	check, claimed, err := r.store.ClaimDue(ctx, r.config.WorkerID, r.config.Lease)
	if err != nil {
		return scheduled > 0, fmt.Errorf("claim Safeheron routing status check: %w", err)
	}
	if !claimed {
		return scheduled > 0, nil
	}

	checkedAt := r.config.Now().UTC()
	snapshot, lookupErr := r.lookup.LookupTransaction(ctx, safeheron.TransactionLookup{TxKey: check.TxKey})
	if lookupErr != nil {
		return true, r.finishFailure(ctx, check, checkedAt, "PROVIDER_LOOKUP_FAILED", lookupErr)
	}
	if validationErr := validateRoutingStatusSnapshot(check, snapshot); validationErr != nil {
		return true, r.finishFailure(ctx, check, checkedAt, "PROVIDER_SNAPSHOT_INVALID", validationErr)
	}

	eventID := routingStatusSnapshotEventID(*snapshot)
	input := companyfund.OwnedProviderPayloadInput{
		Channel:         companyfund.ChannelSafeheron,
		ProviderEventID: eventID,
		EventType:       companyfund.SafeheronTransactionHistorySnapshotEventType,
		Body:            append([]byte(nil), snapshot.RawPayload...),
	}
	if _, err := r.ingester.Ingest(ctx, input); err != nil {
		return true, r.finishFailure(ctx, check, checkedAt, "ROUTING_EVENT_INGEST_FAILED", err)
	}

	status := strings.ToUpper(strings.TrimSpace(snapshot.TransactionStatus))
	terminal := isTerminalRoutingStatus(status)
	var nextCheckAt *time.Time
	if !terminal {
		next := routingStatusNextCheckAt(check, checkedAt)
		nextCheckAt = &next
	}
	if err := r.store.CompleteObserved(ctx, routingStatusObserved{
		Check: check, CheckedAt: checkedAt, Status: status, EventID: eventID,
		Terminal: terminal, NextCheckAt: nextCheckAt,
	}); err != nil {
		return true, fmt.Errorf("complete Safeheron routing status observation: %w", err)
	}
	r.notifyCompleted(true)
	return true, nil
}

func (r *StatusRefresher) finishFailure(
	ctx context.Context,
	check RoutingStatusCheck,
	checkedAt time.Time,
	code string,
	cause error,
) error {
	next := routingStatusNextCheckAt(check, checkedAt)
	if err := r.store.CompleteFailed(ctx, routingStatusFailure{
		Check: check, CheckedAt: checkedAt, ErrorCode: code, NextCheckAt: next,
	}); err != nil {
		return fmt.Errorf("complete Safeheron routing status failure %s: %w", code, err)
	}
	r.notifyCompleted(false)
	return fmt.Errorf("Safeheron routing status check %s: %w", code, cause)
}

func (r *StatusRefresher) notifyCompleted(snapshotStored bool) {
	if snapshotStored && r.onSnapshotStored != nil {
		r.onSnapshotStored()
	}
	if r.onCheckCompleted != nil {
		r.onCheckCompleted()
	}
}

func routingStatusNextCheckAt(check RoutingStatusCheck, now time.Time) time.Time {
	index := check.AttemptCount
	if index < 0 {
		index = 0
	}
	if index >= len(defaultRoutingStatusSchedule) {
		return now.Add(defaultRoutingStatusSchedule[len(defaultRoutingStatusSchedule)-1])
	}
	next := check.FirstSeenAt.Add(defaultRoutingStatusSchedule[index])
	if !next.After(now) {
		return now.Add(defaultRoutingStatusInitialDelay)
	}
	return next
}

func validateRoutingStatusSnapshot(check RoutingStatusCheck, snapshot *safeheron.TransactionSnapshot) error {
	if snapshot == nil || strings.TrimSpace(check.TxKey) == "" || strings.TrimSpace(snapshot.TxKey) != strings.TrimSpace(check.TxKey) {
		return fmt.Errorf("Safeheron routing status snapshot transaction identity does not match")
	}
	status := strings.ToUpper(strings.TrimSpace(snapshot.TransactionStatus))
	switch status {
	case "SUBMITTED", "SIGNING", "BROADCASTING", "CONFIRMING", "COMPLETED", "FAILED", "CANCELLED", "REJECTED":
	default:
		return fmt.Errorf("Safeheron routing status snapshot has unsupported status")
	}
	if len(snapshot.RawPayload) == 0 || !json.Valid(snapshot.RawPayload) {
		return fmt.Errorf("Safeheron routing status snapshot canonical payload is invalid")
	}
	return nil
}

func routingStatusSnapshotEventID(snapshot safeheron.TransactionSnapshot) string {
	digest := sha256.New()
	for _, value := range []string{
		"safeheron-routing-status", "v1", strings.TrimSpace(snapshot.TxKey),
		strings.ToUpper(strings.TrimSpace(snapshot.TransactionStatus)), string(snapshot.RawPayload),
	} {
		_, _ = fmt.Fprintf(digest, "%d:%s", len(value), value)
	}
	return "safeheron-routing-status:v1:" + hex.EncodeToString(digest.Sum(nil))
}

func (r *StatusRefresher) Run(ctx context.Context) {
	if r != nil && r.runner != nil {
		r.runner.Run(ctx)
	}
}

var _ RoutingStatusLookup = (safeheron.TransactionHistoryClient)(nil)
