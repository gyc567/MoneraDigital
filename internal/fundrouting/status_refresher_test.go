package fundrouting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"monera-digital/internal/companyfund"
	"monera-digital/internal/safeheron"
)

type routingStatusCheckStoreStub struct {
	scheduled           int64
	scheduleErr         error
	check               RoutingStatusCheck
	claimed             bool
	claimErr            error
	nextDue             time.Time
	observed            []routingStatusObserved
	completeObservedErr error
	failed              []routingStatusFailure
	completeFailedErr   error
}

func (s *routingStatusCheckStoreStub) ScheduleOpen(context.Context, time.Duration) (int64, error) {
	return s.scheduled, s.scheduleErr
}

func (s *routingStatusCheckStoreStub) ClaimDue(context.Context, string, time.Duration) (RoutingStatusCheck, bool, error) {
	return s.check, s.claimed, s.claimErr
}

func (s *routingStatusCheckStoreStub) CompleteObserved(_ context.Context, observed routingStatusObserved) error {
	s.observed = append(s.observed, observed)
	return s.completeObservedErr
}

func (s *routingStatusCheckStoreStub) CompleteFailed(_ context.Context, failure routingStatusFailure) error {
	s.failed = append(s.failed, failure)
	return s.completeFailedErr
}

func (s *routingStatusCheckStoreStub) NextDue(context.Context, time.Duration) (time.Time, error) {
	return s.nextDue, nil
}

type routingStatusLookupStub struct {
	snapshot *safeheron.TransactionSnapshot
	err      error
	lookups  []safeheron.TransactionLookup
}

func (s *routingStatusLookupStub) LookupTransaction(_ context.Context, lookup safeheron.TransactionLookup) (*safeheron.TransactionSnapshot, error) {
	s.lookups = append(s.lookups, lookup)
	return s.snapshot, s.err
}

type routingStatusIngesterStub struct {
	inputs []companyfund.OwnedProviderPayloadInput
	result companyfund.ProviderEventInsertResult
	err    error
}

func (s *routingStatusIngesterStub) Ingest(_ context.Context, input companyfund.OwnedProviderPayloadInput) (companyfund.ProviderEventInsertResult, error) {
	s.inputs = append(s.inputs, input)
	return s.result, s.err
}

func TestStatusRefresherStoresTerminalLookupAsCanonicalRoutingEvent(t *testing.T) {
	firstSeen := time.Date(2026, 8, 10, 3, 0, 0, 0, time.UTC)
	now := firstSeen.Add(time.Hour)
	store := &routingStatusCheckStoreStub{claimed: true, check: RoutingStatusCheck{
		TxKey: "tx-terminal", FirstSeenAt: firstSeen, AttemptCount: 4, LeaseOwner: "worker-1",
	}}
	lookup := &routingStatusLookupStub{snapshot: routingStatusSnapshot("tx-terminal", "COMPLETED")}
	ingester := &routingStatusIngesterStub{result: companyfund.ProviderEventInsertResult{ID: 41, Inserted: true}}
	refresher, err := NewStatusRefresher(store, lookup, ingester, StatusRefresherConfig{
		WorkerID: "worker-1", Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	stored := 0
	checked := 0
	refresher.SetOnSnapshotStored(func() { stored++ })
	refresher.SetOnCheckCompleted(func() { checked++ })

	worked, err := refresher.ProcessOne(context.Background())
	if err != nil || !worked {
		t.Fatalf("ProcessOne() = %v, %v", worked, err)
	}
	if len(lookup.lookups) != 1 || lookup.lookups[0].TxKey != "tx-terminal" || lookup.lookups[0].CustomerRefID != "" {
		t.Fatalf("lookups = %#v", lookup.lookups)
	}
	if len(ingester.inputs) != 1 || ingester.inputs[0].Channel != companyfund.ChannelSafeheron ||
		ingester.inputs[0].EventType != companyfund.SafeheronTransactionHistorySnapshotEventType ||
		!strings.HasPrefix(ingester.inputs[0].ProviderEventID, "safeheron-routing-status:v1:") ||
		string(ingester.inputs[0].Body) != string(lookup.snapshot.RawPayload) {
		t.Fatalf("ingested input = %#v", ingester.inputs)
	}
	if len(store.observed) != 1 || store.observed[0].Status != "COMPLETED" || store.observed[0].EventID == "" ||
		!store.observed[0].Terminal || store.observed[0].NextCheckAt != nil {
		t.Fatalf("observed completion = %#v", store.observed)
	}
	if stored != 1 || checked != 1 {
		t.Fatalf("callbacks stored=%d checked=%d", stored, checked)
	}
}

func TestStatusRefresherUsesAcceptedNonTerminalSchedule(t *testing.T) {
	firstSeen := time.Date(2026, 8, 10, 3, 0, 0, 0, time.UTC)
	tests := []struct {
		attempt int
		now     time.Time
		want    time.Time
	}{
		{attempt: 1, now: firstSeen.Add(5 * time.Minute), want: firstSeen.Add(15 * time.Minute)},
		{attempt: 2, now: firstSeen.Add(15 * time.Minute), want: firstSeen.Add(30 * time.Minute)},
		{attempt: 3, now: firstSeen.Add(30 * time.Minute), want: firstSeen.Add(time.Hour)},
		{attempt: 4, now: firstSeen.Add(time.Hour), want: firstSeen.Add(2 * time.Hour)},
	}
	for _, testCase := range tests {
		t.Run(testCase.want.String(), func(t *testing.T) {
			store := &routingStatusCheckStoreStub{claimed: true, check: RoutingStatusCheck{
				TxKey: "tx-pending", FirstSeenAt: firstSeen, AttemptCount: testCase.attempt, LeaseOwner: "worker-1",
			}}
			lookup := &routingStatusLookupStub{snapshot: routingStatusSnapshot("tx-pending", "SUBMITTED")}
			refresher, err := NewStatusRefresher(store, lookup, &routingStatusIngesterStub{}, StatusRefresherConfig{
				WorkerID: "worker-1", Now: func() time.Time { return testCase.now },
			})
			if err != nil {
				t.Fatal(err)
			}
			if worked, err := refresher.ProcessOne(context.Background()); err != nil || !worked {
				t.Fatalf("ProcessOne() = %v, %v", worked, err)
			}
			if len(store.observed) != 1 || store.observed[0].Terminal || store.observed[0].NextCheckAt == nil || !store.observed[0].NextCheckAt.Equal(testCase.want) {
				t.Fatalf("observed = %#v, want next %s", store.observed, testCase.want)
			}
		})
	}
}

func TestStatusRefresherPersistsBoundedFailureAndRetries(t *testing.T) {
	firstSeen := time.Date(2026, 8, 10, 3, 0, 0, 0, time.UTC)
	now := firstSeen.Add(5 * time.Minute)
	store := &routingStatusCheckStoreStub{claimed: true, check: RoutingStatusCheck{
		TxKey: "tx-error", FirstSeenAt: firstSeen, AttemptCount: 1, LeaseOwner: "worker-1",
	}}
	lookup := &routingStatusLookupStub{err: errors.New("provider response contained sensitive detail")}
	refresher, err := NewStatusRefresher(store, lookup, &routingStatusIngesterStub{}, StatusRefresherConfig{
		WorkerID: "worker-1", Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	worked, err := refresher.ProcessOne(context.Background())
	if err == nil || !worked {
		t.Fatalf("ProcessOne() = %v, %v", worked, err)
	}
	if len(store.failed) != 1 || store.failed[0].ErrorCode != "PROVIDER_LOOKUP_FAILED" ||
		!store.failed[0].NextCheckAt.Equal(firstSeen.Add(15*time.Minute)) {
		t.Fatalf("failure = %#v", store.failed)
	}
	if strings.Contains(store.failed[0].ErrorCode, "sensitive") {
		t.Fatal("durable failure code leaked provider detail")
	}
}

func TestStatusRefresherRejectsInvalidProviderSnapshotsBeforeIngest(t *testing.T) {
	tests := []struct {
		name     string
		snapshot *safeheron.TransactionSnapshot
	}{
		{name: "missing snapshot", snapshot: nil},
		{name: "wrong transaction", snapshot: routingStatusSnapshot("another-tx", "SUBMITTED")},
		{name: "unsupported status", snapshot: routingStatusSnapshot("tx-invalid", "UNKNOWN")},
		{name: "invalid canonical JSON", snapshot: func() *safeheron.TransactionSnapshot {
			snapshot := routingStatusSnapshot("tx-invalid", "SUBMITTED")
			snapshot.RawPayload = []byte(`{"broken"`)
			return snapshot
		}()},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			store := &routingStatusCheckStoreStub{claimed: true, check: RoutingStatusCheck{
				TxKey: "tx-invalid", FirstSeenAt: time.Now().Add(-time.Hour), AttemptCount: 1, LeaseOwner: "worker-1",
			}}
			ingester := &routingStatusIngesterStub{}
			refresher, err := NewStatusRefresher(store, &routingStatusLookupStub{snapshot: testCase.snapshot}, ingester, StatusRefresherConfig{WorkerID: "worker-1"})
			if err != nil {
				t.Fatal(err)
			}
			worked, processErr := refresher.ProcessOne(context.Background())
			if processErr == nil || !worked {
				t.Fatalf("ProcessOne() = %v, %v", worked, processErr)
			}
			if len(ingester.inputs) != 0 || len(store.failed) != 1 || store.failed[0].ErrorCode != "PROVIDER_SNAPSHOT_INVALID" {
				t.Fatalf("ingested=%d failures=%#v", len(ingester.inputs), store.failed)
			}
		})
	}
}

func TestStatusRefresherPersistsIngestFailureAndDoesNotCompleteObservation(t *testing.T) {
	store := &routingStatusCheckStoreStub{claimed: true, check: RoutingStatusCheck{
		TxKey: "tx-ingest", FirstSeenAt: time.Now().Add(-time.Hour), AttemptCount: 1, LeaseOwner: "worker-1",
	}}
	refresher, err := NewStatusRefresher(store, &routingStatusLookupStub{
		snapshot: routingStatusSnapshot("tx-ingest", "SUBMITTED"),
	}, &routingStatusIngesterStub{err: errors.New("inbox unavailable")}, StatusRefresherConfig{WorkerID: "worker-1"})
	if err != nil {
		t.Fatal(err)
	}
	worked, processErr := refresher.ProcessOne(context.Background())
	if processErr == nil || !worked || len(store.observed) != 0 || len(store.failed) != 1 || store.failed[0].ErrorCode != "ROUTING_EVENT_INGEST_FAILED" {
		t.Fatalf("ProcessOne=%v/%v observed=%#v failed=%#v", worked, processErr, store.observed, store.failed)
	}
}

func TestStatusRefresherPropagatesDurableStoreFailures(t *testing.T) {
	tests := []struct {
		name  string
		store *routingStatusCheckStoreStub
	}{
		{name: "schedule", store: &routingStatusCheckStoreStub{scheduleErr: errors.New("schedule failed")}},
		{name: "claim", store: &routingStatusCheckStoreStub{claimErr: errors.New("claim failed")}},
		{name: "complete observed", store: &routingStatusCheckStoreStub{
			claimed: true, check: RoutingStatusCheck{TxKey: "tx-store", FirstSeenAt: time.Now().Add(-time.Hour), AttemptCount: 1, LeaseOwner: "worker-1"},
			completeObservedErr: errors.New("lease lost"),
		}},
		{name: "complete failure", store: &routingStatusCheckStoreStub{
			claimed: true, check: RoutingStatusCheck{TxKey: "tx-store", FirstSeenAt: time.Now().Add(-time.Hour), AttemptCount: 1, LeaseOwner: "worker-1"},
			completeFailedErr: errors.New("lease lost"),
		}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			lookup := &routingStatusLookupStub{snapshot: routingStatusSnapshot("tx-store", "SUBMITTED")}
			if testCase.name == "complete failure" {
				lookup.err = errors.New("provider unavailable")
			}
			refresher, err := NewStatusRefresher(testCase.store, lookup, &routingStatusIngesterStub{}, StatusRefresherConfig{WorkerID: "worker-1"})
			if err != nil {
				t.Fatal(err)
			}
			if _, processErr := refresher.ProcessOne(context.Background()); processErr == nil {
				t.Fatal("ProcessOne() error = nil, want durable store failure")
			}
		})
	}
}

func routingStatusSnapshot(txKey, status string) *safeheron.TransactionSnapshot {
	raw := []byte(`{"txKey":"` + txKey + `","txHash":"0xabc","coinKey":"USDT_ERC20","txAmount":"12.34","sourceAddress":"0xsource","destinationAddress":"0xdestination","transactionDirection":"OUTFLOW","transactionStatus":"` + status + `","createTime":1786330800000}`)
	return &safeheron.TransactionSnapshot{
		TxKey: txKey, TxHash: "0xabc", CoinKey: "USDT_ERC20", TxAmount: "12.34",
		SourceAddress: "0xsource", DestinationAddress: "0xdestination",
		TransactionDirection: "OUTFLOW", TransactionStatus: status,
		CreateTime: 1786330800000, RawPayload: raw,
	}
}
