package companyfund

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

const maxSafeheronAssetRecognitionRepairBatch = 100

// SafeheronUnrecognizedAssetCandidate is the minimum immutable evidence needed
// to revisit a row that was persisted while the provider coin catalog was not
// available. Amounts, transaction identity, counterparties and finance-owned
// fields are deliberately absent from this repair surface.
type SafeheronUnrecognizedAssetCandidate struct {
	TransactionID    int64
	ProviderAssetKey string
}

// SafeheronAssetRecognitionPatch carries an exact provider CoinKey match. The
// store must apply it only while the row is still an unrecognized Safeheron
// movement with the same provider asset key.
type SafeheronAssetRecognitionPatch struct {
	TransactionID            int64
	ExpectedProviderAssetKey string
	Asset                    AssetIdentity
}

type SafeheronAssetRecognitionRepairStore interface {
	ListSafeheronUnrecognizedAssetCandidates(
		ctx context.Context,
		afterID int64,
		limit int,
	) ([]SafeheronUnrecognizedAssetCandidate, error)
	ApplySafeheronAssetRecognition(ctx context.Context, patch SafeheronAssetRecognitionPatch) (bool, error)
}

type SafeheronAssetRecognitionRepairResult struct {
	Scanned      int
	Repaired     int
	Unrecognized int
	MoreWork     bool
}

// SafeheronAssetRecognitionRepairer advances through bounded ID windows. Its
// process-local cursor is only an efficiency hint: PostgreSQL predicates make
// every patch idempotent and a later full pass begins again at zero.
type SafeheronAssetRecognitionRepairer struct {
	store   SafeheronAssetRecognitionRepairStore
	catalog SafeheronCoinLookup

	mu      sync.Mutex
	afterID int64
}

func NewSafeheronAssetRecognitionRepairer(
	store SafeheronAssetRecognitionRepairStore,
	catalog SafeheronCoinLookup,
) (*SafeheronAssetRecognitionRepairer, error) {
	if store == nil {
		return nil, fmt.Errorf("Safeheron asset recognition repair store is required")
	}
	if catalog == nil {
		return nil, fmt.Errorf("Safeheron asset recognition repair catalog is required")
	}
	return &SafeheronAssetRecognitionRepairer{store: store, catalog: catalog}, nil
}

func (repairer *SafeheronAssetRecognitionRepairer) Sweep(
	ctx context.Context,
	limit int,
) (SafeheronAssetRecognitionRepairResult, error) {
	if repairer == nil || repairer.store == nil || repairer.catalog == nil {
		return SafeheronAssetRecognitionRepairResult{}, fmt.Errorf("Safeheron asset recognition repairer is not configured")
	}
	if limit < 1 || limit > maxSafeheronAssetRecognitionRepairBatch {
		return SafeheronAssetRecognitionRepairResult{}, fmt.Errorf(
			"Safeheron asset recognition repair limit must be between 1 and %d",
			maxSafeheronAssetRecognitionRepairBatch,
		)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return SafeheronAssetRecognitionRepairResult{}, err
	}

	repairer.mu.Lock()
	defer repairer.mu.Unlock()
	candidates, err := repairer.store.ListSafeheronUnrecognizedAssetCandidates(ctx, repairer.afterID, limit)
	if err != nil {
		return SafeheronAssetRecognitionRepairResult{}, fmt.Errorf("list Safeheron unrecognized asset candidates: %w", err)
	}
	result := SafeheronAssetRecognitionRepairResult{Scanned: len(candidates)}
	if len(candidates) == 0 {
		repairer.afterID = 0
		return result, nil
	}

	for _, candidate := range candidates {
		if candidate.TransactionID <= repairer.afterID || strings.TrimSpace(candidate.ProviderAssetKey) == "" ||
			candidate.ProviderAssetKey != strings.TrimSpace(candidate.ProviderAssetKey) {
			return result, fmt.Errorf("invalid Safeheron unrecognized asset candidate")
		}
		coin, lookupErr := repairer.catalog.Lookup(candidate.ProviderAssetKey)
		var coldMiss *SafeheronCoinCatalogColdMissError
		switch {
		case errors.As(lookupErr, &coldMiss):
			return result, lookupErr
		case errors.Is(lookupErr, ErrSafeheronCoinNotFound):
			result.Unrecognized++
			continue
		case lookupErr != nil:
			return result, fmt.Errorf("resolve Safeheron repair asset: %w", lookupErr)
		}
		asset, assetErr := safeheronCatalogAssetIdentity(coin)
		if assetErr != nil {
			return result, assetErr
		}
		asset, assetErr = normalizeSafeheronAssetMapping(candidate.ProviderAssetKey, SafeheronAssetMapping{
			CoinKey: candidate.ProviderAssetKey,
			Asset:   asset,
		}, "repair")
		if assetErr != nil {
			return result, assetErr
		}
		applied, applyErr := repairer.store.ApplySafeheronAssetRecognition(ctx, SafeheronAssetRecognitionPatch{
			TransactionID:            candidate.TransactionID,
			ExpectedProviderAssetKey: candidate.ProviderAssetKey,
			Asset:                    asset,
		})
		if applyErr != nil {
			return result, fmt.Errorf("apply Safeheron asset recognition repair: %w", applyErr)
		}
		if applied {
			result.Repaired++
		}
	}
	repairer.afterID = candidates[len(candidates)-1].TransactionID
	result.MoreWork = len(candidates) == limit
	return result, nil
}
