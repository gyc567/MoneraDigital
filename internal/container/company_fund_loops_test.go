package container

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"monera-digital/internal/companyfund"
	"monera-digital/internal/safeheron"
)

type companyFundLoopRateRefresherStub struct {
	calls  atomic.Int64
	notify chan struct{}
}

func (stub *companyFundLoopRateRefresherStub) Refresh(context.Context) (companyfund.CoinGeckoCurrentRateRefreshResult, error) {
	stub.calls.Add(1)
	select {
	case stub.notify <- struct{}{}:
	default:
	}
	return companyfund.CoinGeckoCurrentRateRefreshResult{}, nil
}

type companyFundLoopValuationSweeperStub struct {
	calls  atomic.Int64
	batch  atomic.Int64
	notify chan struct{}
}

type companyFundLoopRecognitionRepairerStub struct {
	calls  atomic.Int64
	batch  atomic.Int64
	notify chan struct{}
	result companyfund.SafeheronAssetRecognitionRepairResult
	err    error
}

type companyFundLoopRecognitionStoreStub struct {
	listed chan struct{}
}

func (stub *companyFundLoopRecognitionStoreStub) ListSafeheronUnrecognizedAssetCandidates(
	context.Context,
	int64,
	int,
) ([]companyfund.SafeheronUnrecognizedAssetCandidate, error) {
	select {
	case stub.listed <- struct{}{}:
	default:
	}
	return nil, nil
}

func (*companyFundLoopRecognitionStoreStub) ApplySafeheronAssetRecognition(
	context.Context,
	companyfund.SafeheronAssetRecognitionPatch,
) (bool, error) {
	return false, nil
}

type companyFundLoopCoinListerStub struct{}

func (companyFundLoopCoinListerStub) ListCoin(context.Context) ([]safeheron.Coin, error) {
	return []safeheron.Coin{{
		CoinKey: "USDT_ERC20", Symbol: "USDT", BlockChain: "ETHEREUM", BlockchainType: "EVM",
	}}, nil
}

func (stub *companyFundLoopRecognitionRepairerStub) Sweep(
	_ context.Context,
	batchSize int,
) (companyfund.SafeheronAssetRecognitionRepairResult, error) {
	stub.calls.Add(1)
	stub.batch.Store(int64(batchSize))
	select {
	case stub.notify <- struct{}{}:
	default:
	}
	return stub.result, stub.err
}

func (stub *companyFundLoopValuationSweeperStub) Sweep(_ context.Context, batchSize int) companyfund.CompanyFundValuationSweepResult {
	stub.calls.Add(1)
	stub.batch.Store(int64(batchSize))
	select {
	case stub.notify <- struct{}{}:
	default:
	}
	return companyfund.CompanyFundValuationSweepResult{}
}

func TestCompanyFundCurrentValuationLoops_UseIndependentRefreshAndSweepIntervals(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	refresher := &companyFundLoopRateRefresherStub{notify: make(chan struct{}, 4)}
	valuator := &companyFundLoopValuationSweeperStub{notify: make(chan struct{}, 8)}
	refreshDone := make(chan struct{})
	sweepDone := make(chan struct{})
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Errorf("rate refresh loop panicked: %v", recovered)
			}
			close(refreshDone)
		}()
		runCompanyFundCurrentRateRefreshLoop(ctx, refresher, time.Hour)
	}()
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Errorf("valuation sweep loop panicked: %v", recovered)
			}
			close(sweepDone)
		}()
		runCompanyFundCurrentValuationSweepLoop(ctx, valuator, 10*time.Millisecond, 37)
	}()

	select {
	case <-refresher.notify:
	case <-time.After(time.Second):
		t.Fatal("rate refresh loop did not run immediately")
	}
	for count := 0; count < 3; count++ {
		select {
		case <-valuator.notify:
		case <-time.After(time.Second):
			t.Fatalf("valuation sweep %d did not run on its independent ticker", count+1)
		}
	}
	if got := refresher.calls.Load(); got != 1 {
		t.Fatalf("rate refresh calls = %d, want one; valuation sweeps must not induce provider refreshes", got)
	}
	if got := valuator.calls.Load(); got < 3 || valuator.batch.Load() != 37 {
		t.Fatalf("valuation sweeps=%d batch=%d, want at least 3 and 37", got, valuator.batch.Load())
	}

	cancel()
	select {
	case <-refreshDone:
	case <-time.After(time.Second):
		t.Fatal("rate refresh loop did not stop")
	}
	select {
	case <-sweepDone:
	case <-time.After(time.Second):
		t.Fatal("valuation sweep loop did not stop")
	}
}

func TestCompanyFundRateRefreshSuccessWakesValuationSweep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	refresher := &companyFundLoopRateRefresherStub{notify: make(chan struct{}, 2)}
	valuator := &companyFundLoopValuationSweeperStub{notify: make(chan struct{}, 3)}
	valuationLoop := newCompanyFundCurrentValuationSweepLoop(valuator, time.Hour, 10)
	rateLoop := newCompanyFundCurrentRateRefreshLoop(refresher, time.Hour, func() {
		_ = valuationLoop.Notify()
	})
	valuationLoop.Start(ctx)
	defer valuationLoop.Stop()
	select {
	case <-valuator.notify:
	case <-time.After(time.Second):
		t.Fatal("valuation startup sweep did not run")
	}
	rateLoop.Start(ctx)
	defer rateLoop.Stop()

	select {
	case <-refresher.notify:
	case <-time.After(time.Second):
		t.Fatal("rate refresh did not run")
	}
	deadline := time.After(time.Second)
	for valuator.calls.Load() < 2 {
		select {
		case <-valuator.notify:
		case <-deadline:
			t.Fatalf("successful refresh did not wake valuation; calls=%d", valuator.calls.Load())
		}
	}
	cancel()
	rateLoop.Stop()
	valuationLoop.Stop()
}

func TestCompanyFundSafeheronRecognitionRepairWakesValuationAfterRepair(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repairer := &companyFundLoopRecognitionRepairerStub{
		notify: make(chan struct{}, 1),
		result: companyfund.SafeheronAssetRecognitionRepairResult{
			Scanned: 2, Repaired: 1, Unrecognized: 1,
		},
	}
	valuationWake := make(chan struct{}, 1)
	loop := newCompanyFundSafeheronAssetRecognitionRepairLoop(repairer, time.Hour, 37, func() {
		valuationWake <- struct{}{}
	})
	loop.Start(ctx)
	defer loop.Stop()

	select {
	case <-repairer.notify:
	case <-time.After(time.Second):
		t.Fatal("Safeheron recognition repair did not run immediately")
	}
	select {
	case <-valuationWake:
	case <-time.After(time.Second):
		t.Fatal("successful asset repair did not wake valuation")
	}
	if repairer.calls.Load() != 1 || repairer.batch.Load() != 37 {
		t.Fatalf("repair calls=%d batch=%d", repairer.calls.Load(), repairer.batch.Load())
	}
}

func TestCompanyFundSafeheronRecognitionRepairLoopContainsFailures(t *testing.T) {
	for _, testCase := range []struct {
		name string
		err  error
	}{
		{name: "cold catalog", err: &companyfund.SafeheronCoinCatalogColdMissError{CoinKey: "USDT_ERC20"}},
		{name: "database", err: errors.New("database failed")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			repairer := &companyFundLoopRecognitionRepairerStub{
				notify: make(chan struct{}, 1),
				result: companyfund.SafeheronAssetRecognitionRepairResult{Scanned: 1},
				err:    testCase.err,
			}
			loop := newCompanyFundSafeheronAssetRecognitionRepairLoop(repairer, time.Hour, 1, nil)
			loop.Start(ctx)
			defer loop.Stop()
			select {
			case <-repairer.notify:
			case <-time.After(time.Second):
				t.Fatal("failed repair cycle did not run")
			}
		})
	}
	if loop := newCompanyFundSafeheronAssetRecognitionRepairLoop(
		&companyFundLoopRecognitionRepairerStub{},
		-time.Second,
		1,
		nil,
	); loop != nil {
		t.Fatal("invalid repair interval created a loop")
	}
}

func TestStartCompanyFundAuxiliaryLoopsRunsRecognitionRepairWithoutOtherAdapters(t *testing.T) {
	store := &companyFundLoopRecognitionStoreStub{listed: make(chan struct{}, 1)}
	catalog, err := companyfund.NewSafeheronCoinCatalog(companyFundLoopCoinListerStub{}, companyfund.SafeheronCoinCatalogConfig{})
	if err != nil || catalog.Refresh(t.Context()) != nil {
		t.Fatalf("catalog setup: %v", err)
	}
	repairer, err := companyfund.NewSafeheronAssetRecognitionRepairer(store, catalog)
	if err != nil {
		t.Fatal(err)
	}
	container := &Container{CompanyFundSafeheronAssetRepairer: repairer}
	startCompanyFundAuxiliaryLoops(container, t.Context(), companyFundRuntimeConfig{
		SafeheronAssetRepairInterval:  time.Hour,
		SafeheronAssetRepairBatchSize: 1,
	}, nil, nil, nil)
	t.Cleanup(func() { stopCompanyFundAuxiliaryLoops(container) })

	select {
	case <-store.listed:
	case <-time.After(time.Second):
		t.Fatal("recognition-only auxiliary runtime did not scan immediately")
	}
	if container.companyFundSafeheronAssetRepairLoop == nil || container.companyFundAuxDone == nil {
		t.Fatal("recognition repair loop was not retained for lifecycle shutdown")
	}

	invalid := &Container{CompanyFundSafeheronAssetRepairer: repairer}
	startCompanyFundAuxiliaryLoops(invalid, t.Context(), companyFundRuntimeConfig{
		SafeheronAssetRepairInterval: -time.Second,
	}, nil, nil, nil)
	if invalid.companyFundAuxDone != nil {
		stopCompanyFundAuxiliaryLoops(invalid)
		t.Fatal("invalid recognition repair config started auxiliary lifecycle")
	}
}
