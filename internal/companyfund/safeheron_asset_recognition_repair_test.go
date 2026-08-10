package companyfund

import (
	"context"
	"errors"
	"testing"

	"monera-digital/internal/safeheron"
)

type safeheronAssetRecognitionRepairStoreStub struct {
	candidates []SafeheronUnrecognizedAssetCandidate
	listErr    error
	applyErr   error
	patches    []SafeheronAssetRecognitionPatch
	afterIDs   []int64
	notApplied bool
}

func (store *safeheronAssetRecognitionRepairStoreStub) ListSafeheronUnrecognizedAssetCandidates(
	_ context.Context,
	afterID int64,
	limit int,
) ([]SafeheronUnrecognizedAssetCandidate, error) {
	store.afterIDs = append(store.afterIDs, afterID)
	if store.listErr != nil {
		return nil, store.listErr
	}
	result := make([]SafeheronUnrecognizedAssetCandidate, 0, limit)
	for _, candidate := range store.candidates {
		if candidate.TransactionID > afterID {
			result = append(result, candidate)
		}
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

func (store *safeheronAssetRecognitionRepairStoreStub) ApplySafeheronAssetRecognition(
	_ context.Context,
	patch SafeheronAssetRecognitionPatch,
) (bool, error) {
	store.patches = append(store.patches, patch)
	if store.applyErr != nil {
		return false, store.applyErr
	}
	return !store.notApplied, nil
}

type safeheronCoinLookupStub struct {
	coin safeheron.Coin
	err  error
}

func (stub safeheronCoinLookupStub) Lookup(string) (safeheron.Coin, error) {
	return stub.coin, stub.err
}

func TestSafeheronAssetRecognitionRepairerRepairsExactHitsAndLeavesTrueUnknowns(t *testing.T) {
	store := &safeheronAssetRecognitionRepairStoreStub{candidates: []SafeheronUnrecognizedAssetCandidate{
		{TransactionID: 385, ProviderAssetKey: "USDT_ERC20"},
		{TransactionID: 386, ProviderAssetKey: "UNKNOWN_EXACT"},
	}}
	catalog, err := NewSafeheronCoinCatalog(&fakeSafeheronCoinLister{coins: []safeheron.Coin{{
		CoinKey: "USDT_ERC20", Symbol: "USDT", BlockChain: "ETHEREUM", BlockchainType: "EVM",
		TokenIdentifier: "0xDaC17F",
	}}}, SafeheronCoinCatalogConfig{})
	if err != nil || catalog.Refresh(t.Context()) != nil {
		t.Fatalf("catalog setup: %v", err)
	}
	repairer, err := NewSafeheronAssetRecognitionRepairer(store, catalog)
	if err != nil {
		t.Fatal(err)
	}

	result, err := repairer.Sweep(t.Context(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 2 || result.Repaired != 1 || result.Unrecognized != 1 || result.MoreWork {
		t.Fatalf("repair result = %#v", result)
	}
	if len(store.patches) != 1 {
		t.Fatalf("patches = %#v", store.patches)
	}
	patch := store.patches[0]
	if patch.TransactionID != 385 || patch.ExpectedProviderAssetKey != "USDT_ERC20" ||
		patch.Asset.Currency != "USDT" || patch.Asset.ChainCode != "ETHEREUM" ||
		patch.Asset.ProviderAssetKey != "USDT_ERC20" || patch.Asset.ContractAddress != "0xdac17f" {
		t.Fatalf("repair patch = %#v", patch)
	}
}

func TestSafeheronAssetRecognitionRepairerColdCatalogDefersWithoutWriting(t *testing.T) {
	store := &safeheronAssetRecognitionRepairStoreStub{candidates: []SafeheronUnrecognizedAssetCandidate{{
		TransactionID: 385, ProviderAssetKey: "USDT_ERC20",
	}}}
	catalog, err := NewSafeheronCoinCatalog(&fakeSafeheronCoinLister{}, SafeheronCoinCatalogConfig{})
	if err != nil {
		t.Fatal(err)
	}
	repairer, err := NewSafeheronAssetRecognitionRepairer(store, catalog)
	if err != nil {
		t.Fatal(err)
	}

	result, err := repairer.Sweep(t.Context(), 10)
	var coldMiss *SafeheronCoinCatalogColdMissError
	if !errors.As(err, &coldMiss) || result.Repaired != 0 || len(store.patches) != 0 {
		t.Fatalf("cold repair result=%#v err=%v patches=%#v", result, err, store.patches)
	}
}

func TestSafeheronAssetRecognitionRepairerUsesBoundedCursorWindows(t *testing.T) {
	store := &safeheronAssetRecognitionRepairStoreStub{candidates: []SafeheronUnrecognizedAssetCandidate{
		{TransactionID: 10, ProviderAssetKey: "USDT_ERC20"},
		{TransactionID: 20, ProviderAssetKey: "USDT_ERC20"},
	}}
	catalog, err := NewSafeheronCoinCatalog(&fakeSafeheronCoinLister{coins: []safeheron.Coin{{
		CoinKey: "USDT_ERC20", Symbol: "USDT", BlockChain: "ETHEREUM", BlockchainType: "EVM",
	}}}, SafeheronCoinCatalogConfig{})
	if err != nil || catalog.Refresh(t.Context()) != nil {
		t.Fatalf("catalog setup: %v", err)
	}
	repairer, err := NewSafeheronAssetRecognitionRepairer(store, catalog)
	if err != nil {
		t.Fatal(err)
	}

	first, err := repairer.Sweep(t.Context(), 1)
	if err != nil || !first.MoreWork || first.Repaired != 1 {
		t.Fatalf("first sweep = %#v, %v", first, err)
	}
	second, err := repairer.Sweep(t.Context(), 1)
	if err != nil || !second.MoreWork || second.Repaired != 1 {
		t.Fatalf("second sweep = %#v, %v", second, err)
	}
	third, err := repairer.Sweep(t.Context(), 1)
	if err != nil || third.MoreWork || third.Scanned != 0 {
		t.Fatalf("third sweep = %#v, %v", third, err)
	}
	if len(store.afterIDs) != 3 || store.afterIDs[0] != 0 || store.afterIDs[1] != 10 || store.afterIDs[2] != 20 {
		t.Fatalf("cursor sequence = %#v", store.afterIDs)
	}
}

func TestSafeheronAssetRecognitionRepairerPropagatesStoreFailuresWithoutAdvancing(t *testing.T) {
	catalog, err := NewSafeheronCoinCatalog(&fakeSafeheronCoinLister{coins: []safeheron.Coin{{
		CoinKey: "USDT_ERC20", Symbol: "USDT", BlockChain: "ETHEREUM", BlockchainType: "EVM",
	}}}, SafeheronCoinCatalogConfig{})
	if err != nil || catalog.Refresh(t.Context()) != nil {
		t.Fatalf("catalog setup: %v", err)
	}

	t.Run("list", func(t *testing.T) {
		storeErr := errors.New("list failed")
		store := &safeheronAssetRecognitionRepairStoreStub{listErr: storeErr}
		repairer, err := NewSafeheronAssetRecognitionRepairer(store, catalog)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repairer.Sweep(t.Context(), 1); !errors.Is(err, storeErr) {
			t.Fatalf("list failure = %v", err)
		}
	})

	t.Run("apply", func(t *testing.T) {
		storeErr := errors.New("apply failed")
		store := &safeheronAssetRecognitionRepairStoreStub{
			candidates: []SafeheronUnrecognizedAssetCandidate{{TransactionID: 1, ProviderAssetKey: "USDT_ERC20"}},
			applyErr:   storeErr,
		}
		repairer, err := NewSafeheronAssetRecognitionRepairer(store, catalog)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repairer.Sweep(t.Context(), 1); !errors.Is(err, storeErr) {
			t.Fatalf("apply failure = %v", err)
		}
	})
}

func TestSafeheronAssetRecognitionRepairerFailsClosedOnInvalidEvidence(t *testing.T) {
	store := &safeheronAssetRecognitionRepairStoreStub{candidates: []SafeheronUnrecognizedAssetCandidate{{
		TransactionID: 1, ProviderAssetKey: "USDT_ERC20",
	}}}

	t.Run("unexpected lookup failure", func(t *testing.T) {
		lookupErr := errors.New("lookup failed")
		repairer, err := NewSafeheronAssetRecognitionRepairer(store, safeheronCoinLookupStub{err: lookupErr})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repairer.Sweep(t.Context(), 1); !errors.Is(err, lookupErr) {
			t.Fatalf("lookup failure = %v", err)
		}
	})

	t.Run("incomplete provider metadata", func(t *testing.T) {
		repairer, err := NewSafeheronAssetRecognitionRepairer(store, safeheronCoinLookupStub{coin: safeheron.Coin{
			CoinKey: "USDT_ERC20", BlockChain: "ETHEREUM", BlockchainType: "EVM",
		}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repairer.Sweep(t.Context(), 1); err == nil {
			t.Fatal("incomplete provider coin metadata accepted")
		}
	})

	t.Run("invalid persisted candidate", func(t *testing.T) {
		invalidStore := &safeheronAssetRecognitionRepairStoreStub{candidates: []SafeheronUnrecognizedAssetCandidate{{
			TransactionID: 1, ProviderAssetKey: " USDT_ERC20",
		}}}
		repairer, err := NewSafeheronAssetRecognitionRepairer(invalidStore, safeheronCoinLookupStub{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repairer.Sweep(t.Context(), 1); err == nil {
			t.Fatal("invalid persisted candidate accepted")
		}
	})

	t.Run("concurrent repair already won", func(t *testing.T) {
		staleStore := &safeheronAssetRecognitionRepairStoreStub{
			candidates: store.candidates,
			notApplied: true,
		}
		repairer, err := NewSafeheronAssetRecognitionRepairer(staleStore, safeheronCoinLookupStub{coin: safeheron.Coin{
			CoinKey: "USDT_ERC20", Symbol: "USDT", BlockChain: "ETHEREUM", BlockchainType: "EVM",
		}})
		if err != nil {
			t.Fatal(err)
		}
		result, err := repairer.Sweep(t.Context(), 1)
		if err != nil || result.Repaired != 0 || !result.MoreWork {
			t.Fatalf("concurrent repair result = %#v, %v", result, err)
		}
	})
}

func TestNewSafeheronAssetRecognitionRepairerRejectsMissingDependenciesAndInvalidLimit(t *testing.T) {
	catalog, err := NewSafeheronCoinCatalog(&fakeSafeheronCoinLister{}, SafeheronCoinCatalogConfig{})
	if err != nil {
		t.Fatal(err)
	}
	store := &safeheronAssetRecognitionRepairStoreStub{}
	if _, err := NewSafeheronAssetRecognitionRepairer(nil, catalog); err == nil {
		t.Fatal("nil store accepted")
	}
	if _, err := NewSafeheronAssetRecognitionRepairer(store, nil); err == nil {
		t.Fatal("nil catalog accepted")
	}
	repairer, err := NewSafeheronAssetRecognitionRepairer(store, catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{0, maxSafeheronAssetRecognitionRepairBatch + 1} {
		if _, err := repairer.Sweep(t.Context(), limit); err == nil {
			t.Fatalf("limit %d accepted", limit)
		}
	}
	var nilRepairer *SafeheronAssetRecognitionRepairer
	if _, err := nilRepairer.Sweep(t.Context(), 1); err == nil {
		t.Fatal("nil repairer sweep accepted")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repairer.Sweep(canceled, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled sweep error = %v", err)
	}
}
