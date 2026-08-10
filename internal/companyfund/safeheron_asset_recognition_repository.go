package companyfund

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const selectSafeheronUnrecognizedAssetCandidatesSQL = `
SELECT id, provider_asset_key
FROM company_fund_transactions
WHERE id > $1
  AND channel = 'SAFEHERON'
  AND is_unrecognized_asset = true
  AND provider_asset_key IS NOT NULL
  AND btrim(provider_asset_key) <> ''
ORDER BY id
LIMIT $2`

const applySafeheronAssetRecognitionSQL = `
UPDATE company_fund_transactions
SET currency = $3,
    chain_code = $4,
    asset_contract = NULLIF($5, ''),
    is_unrecognized_asset = false,
    updated_at = clock_timestamp()
WHERE id = $1
  AND provider_asset_key = $2
  AND channel = 'SAFEHERON'
  AND is_unrecognized_asset = true
RETURNING id`

func (r *DBRepository) ListSafeheronUnrecognizedAssetCandidates(
	ctx context.Context,
	afterID int64,
	limit int,
) ([]SafeheronUnrecognizedAssetCandidate, error) {
	if afterID < 0 {
		return nil, fmt.Errorf("Safeheron asset recognition repair cursor must be non-negative")
	}
	if limit < 1 || limit > maxSafeheronAssetRecognitionRepairBatch {
		return nil, fmt.Errorf(
			"Safeheron asset recognition repair limit must be between 1 and %d",
			maxSafeheronAssetRecognitionRepairBatch,
		)
	}
	if err := r.requireDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, selectSafeheronUnrecognizedAssetCandidatesSQL, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("query Safeheron unrecognized asset candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]SafeheronUnrecognizedAssetCandidate, 0, limit)
	for rows.Next() {
		var candidate SafeheronUnrecognizedAssetCandidate
		if err := rows.Scan(&candidate.TransactionID, &candidate.ProviderAssetKey); err != nil {
			return nil, fmt.Errorf("scan Safeheron unrecognized asset candidate: %w", err)
		}
		if candidate.TransactionID <= afterID || strings.TrimSpace(candidate.ProviderAssetKey) == "" ||
			candidate.ProviderAssetKey != strings.TrimSpace(candidate.ProviderAssetKey) {
			return nil, fmt.Errorf("invalid persisted Safeheron unrecognized asset candidate")
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Safeheron unrecognized asset candidates: %w", err)
	}
	return candidates, nil
}

func (r *DBRepository) ApplySafeheronAssetRecognition(
	ctx context.Context,
	patch SafeheronAssetRecognitionPatch,
) (bool, error) {
	if patch.TransactionID <= 0 || strings.TrimSpace(patch.ExpectedProviderAssetKey) == "" ||
		patch.ExpectedProviderAssetKey != strings.TrimSpace(patch.ExpectedProviderAssetKey) {
		return false, fmt.Errorf("invalid Safeheron asset recognition patch identity")
	}
	asset, err := normalizeSafeheronAssetMapping(patch.ExpectedProviderAssetKey, SafeheronAssetMapping{
		CoinKey: patch.ExpectedProviderAssetKey,
		Asset:   patch.Asset,
	}, "repair")
	if err != nil {
		return false, err
	}
	if err := r.requireDB(); err != nil {
		return false, err
	}
	var updatedID int64
	err = r.db.QueryRowContext(
		ctx,
		applySafeheronAssetRecognitionSQL,
		patch.TransactionID,
		patch.ExpectedProviderAssetKey,
		asset.Currency,
		asset.ChainCode,
		asset.ContractAddress,
	).Scan(&updatedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("update Safeheron asset recognition: %w", err)
	}
	if updatedID != patch.TransactionID {
		return false, fmt.Errorf("Safeheron asset recognition update returned a different transaction")
	}
	return true, nil
}

var _ SafeheronAssetRecognitionRepairStore = (*DBRepository)(nil)
