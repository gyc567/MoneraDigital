package companyfund

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const selectAirwallexRelationshipTargetSQL = `
SELECT id,
	movement_key,
	movement_kind,
	currency,
	amount::text,
	transaction_direction,
	from_company_fund_account_id,
	to_company_fund_account_id
FROM company_fund_transactions
WHERE channel = 'AIRWALLEX'
  AND provider_account_key = $1
  AND provider_transaction_id = $2
ORDER BY id
LIMIT 2`

const selectAirwallexConversionPeerTaskSQL = `
SELECT id,
	provider_transaction_fact_id,
	subject_provider_transaction_id,
	relationship_reference_type,
	relationship_reference_key,
	relationship_group_key,
	evidence_reference,
	task_payload,
	task_payload_digest
FROM company_fund_ledger_tasks
WHERE channel = 'AIRWALLEX'
  AND provider_account_key = $1
  AND task_kind = 'CONVERSION_PAIR'
  AND relationship_group_key = $2
  AND id <> $3
  AND task_state IN ('WAITING', 'LEASED', 'COMPLETED')
ORDER BY id
LIMIT 2`

const selectAirwallexConversionFactByIDSQL = `
SELECT ` + providerTransactionFactReturnedColumns + `
FROM company_fund_provider_transaction_facts
WHERE id = $1
  AND channel = 'AIRWALLEX'
  AND provider_account_key = $2`

const completeAirwallexConversionGroupSQL = `
WITH locked AS (
	SELECT id, conversion_leg
	FROM company_fund_transactions
	WHERE channel = 'AIRWALLEX'
	  AND provider_account_key = $1
	  AND conversion_group_key = $2
	FOR UPDATE
), validated AS (
	SELECT COUNT(*) AS total,
		COUNT(*) FILTER (WHERE conversion_leg = 'SELL') AS sell_count,
		COUNT(*) FILTER (WHERE conversion_leg = 'BUY') AS buy_count
	FROM locked
), updated AS (
	UPDATE company_fund_transactions transaction
	SET conversion_group_status = 'COMPLETE',
		conversion_pair_transaction_id = (
			SELECT peer.id
			FROM locked peer
			WHERE peer.id <> transaction.id
		),
		updated_at = clock_timestamp()
	FROM validated
	WHERE transaction.id IN (SELECT id FROM locked)
	  AND validated.total = 2
	  AND validated.sell_count = 1
	  AND validated.buy_count = 1
	  AND (
		transaction.conversion_pair_transaction_id IS NULL
		OR transaction.conversion_pair_transaction_id = (
			SELECT peer.id
			FROM locked peer
			WHERE peer.id <> transaction.id
		)
	  )
	RETURNING transaction.id
)
SELECT COUNT(*) FROM updated`

type AirwallexLedgerTaskProcessorConfig struct {
	Owner                 string
	LeaseDuration         time.Duration
	RetryDelay            time.Duration
	FeeClassification     AirwallexFeeClassificationPolicy
	ReversalPolicyVersion string
	MaintenanceBatch      int
	TaskSLA               time.Duration
}

type AirwallexLedgerTaskProcessor struct {
	repository *DBRepository
	config     AirwallexLedgerTaskProcessorConfig
}

type airwallexRelationshipTarget struct {
	ID        int64
	Key       string
	Kind      MovementKind
	Currency  string
	Amount    decimal.Decimal
	Direction Direction
	FromID    *int64
	ToID      *int64
}

func NewAirwallexLedgerTaskProcessor(
	repository *DBRepository,
	config AirwallexLedgerTaskProcessorConfig,
) (*AirwallexLedgerTaskProcessor, error) {
	if repository == nil || repository.db == nil {
		return nil, fmt.Errorf("Airwallex ledger task processor repository is required")
	}
	config.Owner = strings.TrimSpace(config.Owner)
	config.FeeClassification.Level1Code = strings.TrimSpace(config.FeeClassification.Level1Code)
	config.FeeClassification.Level2Code = strings.TrimSpace(config.FeeClassification.Level2Code)
	config.FeeClassification.PolicyVersion = strings.TrimSpace(config.FeeClassification.PolicyVersion)
	config.ReversalPolicyVersion = strings.TrimSpace(config.ReversalPolicyVersion)
	if err := validateLeaseOwner(config.Owner); err != nil {
		return nil, err
	}
	if config.LeaseDuration <= 0 || config.RetryDelay <= 0 {
		return nil, fmt.Errorf("Airwallex ledger task processor durations must be positive")
	}
	if config.MaintenanceBatch == 0 {
		config.MaintenanceBatch = 100
	}
	if config.TaskSLA == 0 {
		config.TaskSLA = 7 * 24 * time.Hour
	}
	if config.MaintenanceBatch < 1 || config.MaintenanceBatch > 1000 || config.TaskSLA <= 0 {
		return nil, fmt.Errorf("Airwallex ledger maintenance bounds are invalid")
	}
	return &AirwallexLedgerTaskProcessor{repository: repository, config: config}, nil
}

func (processor *AirwallexLedgerTaskProcessor) Maintain(ctx context.Context) (bool, error) {
	if processor == nil || processor.repository == nil {
		return false, fmt.Errorf("Airwallex ledger task processor is not configured")
	}
	backfillPolicy := processor.config.FeeClassification
	if backfillPolicy.PolicyVersion == "" {
		backfillPolicy.PolicyVersion = "airwallex-fee-policy-unconfigured-v1"
	}
	enqueued, err := processor.repository.EnqueueAirwallexFeeClassificationBackfill(
		ctx,
		backfillPolicy,
		processor.config.MaintenanceBatch,
		processor.config.TaskSLA,
	)
	if err != nil {
		return false, err
	}
	synchronized := 0
	if processor.validReversalPolicy() {
		var err error
		synchronized, err = processor.repository.SynchronizeReversalClassificationInheritances(
			ctx,
			processor.config.ReversalPolicyVersion,
			processor.config.MaintenanceBatch,
		)
		if err != nil {
			return enqueued > 0, err
		}
	}
	return enqueued > 0 || synchronized > 0, nil
}

func (processor *AirwallexLedgerTaskProcessor) validReversalPolicy() bool {
	return processor != nil &&
		processor.config.ReversalPolicyVersion != "" &&
		len(processor.config.ReversalPolicyVersion) <= 64
}

func (processor *AirwallexLedgerTaskProcessor) ProcessNext(ctx context.Context) (LedgerTaskProcessResult, error) {
	if processor == nil || processor.repository == nil {
		return LedgerTaskProcessResult{}, fmt.Errorf("Airwallex ledger task processor is not configured")
	}
	lease, found, err := processor.repository.ClaimCompanyFundLedgerTask(
		ctx, processor.config.Owner, processor.config.LeaseDuration,
	)
	if err != nil || !found {
		if err != nil {
			return LedgerTaskProcessResult{}, err
		}
		return LedgerTaskProcessResult{Outcome: LedgerTaskProcessIdle}, nil
	}
	result := LedgerTaskProcessResult{TaskID: lease.ID, Kind: lease.Kind}
	transactionID, retryCode, terminalCode, err := processor.processLease(ctx, lease)
	if err != nil {
		return result, err
	}
	if terminalCode != "" {
		if err := processor.repository.deadLetterCompanyFundLedgerTask(
			ctx, lease.ID, processor.config.Owner, lease.AttemptCount, terminalCode,
		); err != nil {
			return result, err
		}
		result.Outcome = LedgerTaskProcessDeadLetter
		return result, nil
	}
	if retryCode != "" {
		outcome, err := processor.repository.retryCompanyFundLedgerTask(
			ctx, lease.ID, processor.config.Owner, lease.AttemptCount, processor.config.RetryDelay, retryCode,
		)
		if err != nil {
			return result, err
		}
		result.Outcome = outcome
		return result, nil
	}
	if err := processor.repository.completeCompanyFundLedgerTask(
		ctx, lease.ID, processor.config.Owner, lease.AttemptCount, transactionID,
	); err != nil {
		return result, err
	}
	result.Outcome = LedgerTaskProcessCompleted
	return result, nil
}

func (processor *AirwallexLedgerTaskProcessor) processLease(
	ctx context.Context,
	lease CompanyFundLedgerTaskLease,
) (*int64, string, string, error) {
	switch lease.Kind {
	case LedgerTaskKindFeeRelationship:
		return processor.processLinkedMovement(ctx, lease, false)
	case LedgerTaskKindReversalRelationship:
		return processor.processLinkedMovement(ctx, lease, true)
	case LedgerTaskKindConversionPair:
		return processor.processConversion(ctx, lease)
	case LedgerTaskKindFeeClassification:
		if lease.SubjectTransactionID == nil {
			return nil, "", "FEE_CLASSIFICATION_SUBJECT_MISSING", nil
		}
		_, err := processor.repository.ApplyAirwallexFeeClassification(
			ctx, *lease.SubjectTransactionID, processor.config.FeeClassification,
		)
		if err != nil {
			return nil, "FEE_CATEGORY_UNAVAILABLE", "", nil
		}
		return lease.SubjectTransactionID, "", "", nil
	case LedgerTaskKindReversalInheritance:
		if lease.SubjectTransactionID == nil {
			return nil, "", "REVERSAL_INHERITANCE_SUBJECT_MISSING", nil
		}
		_, err := processor.repository.ApplyReversalClassificationInheritance(
			ctx, *lease.SubjectTransactionID, processor.config.ReversalPolicyVersion,
		)
		if err != nil {
			return nil, "REVERSAL_INHERITANCE_RETRY", "", nil
		}
		return lease.SubjectTransactionID, "", "", nil
	default:
		return nil, "", "UNSUPPORTED_LEDGER_TASK_KIND", nil
	}
}

func (processor *AirwallexLedgerTaskProcessor) processLinkedMovement(
	ctx context.Context,
	lease CompanyFundLedgerTaskLease,
	reversal bool,
) (*int64, string, string, error) {
	target, found, conflict, err := processor.repository.resolveAirwallexRelationshipTarget(
		ctx, lease.ProviderAccountKey, lease.RelationshipReferenceKey,
	)
	if err != nil {
		return nil, "", "", err
	}
	if conflict {
		return nil, "", "RELATIONSHIP_TARGET_AMBIGUOUS", nil
	}
	if !found {
		return nil, "RELATIONSHIP_TARGET_WAITING", "", nil
	}
	if target.ID <= 0 || target.Key == lease.Proposal.MovementKey {
		return nil, "", "RELATIONSHIP_SELF_REFERENCE", nil
	}
	if reversal && target.Kind == MovementKindReversal {
		return nil, "", "REVERSAL_RELATIONSHIP_CYCLE", nil
	}
	if reversal {
		if conflictCode := validateAirwallexReversalSemantics(lease.Proposal, target); conflictCode != "" {
			return nil, "", conflictCode, nil
		}
	}
	proposal := lease.Proposal
	proposal.ProviderTransactionFactID = &lease.ProviderTransactionFactID
	proposal.RelationshipReferenceType = lease.RelationshipReferenceType
	proposal.RelationshipReferenceKey = lease.RelationshipReferenceKey
	proposal.RelationshipGroupKey = lease.RelationshipGroupKey
	if reversal {
		proposal.ReversalOfMovementKey = target.Key
	} else {
		proposal.ParentMovementKey = target.Key
	}
	upserted, err := processor.repository.UpsertCompanyFundTransaction(ctx, proposal)
	if err != nil {
		var quarantine *TransactionQuarantineError
		if errors.As(err, &quarantine) {
			return nil, "", "RELATIONSHIP_MOVEMENT_CONFLICT", nil
		}
		return nil, "", "", err
	}
	transactionID := upserted.ID
	if reversal {
		if _, err := processor.repository.ApplyReversalClassificationInheritance(
			ctx, transactionID, processor.config.ReversalPolicyVersion,
		); err != nil {
			return &transactionID, "REVERSAL_INHERITANCE_RETRY", "", nil
		}
	} else {
		if _, err := processor.repository.ApplyAirwallexFeeClassification(
			ctx, transactionID, processor.config.FeeClassification,
		); err != nil {
			return &transactionID, "FEE_CATEGORY_UNAVAILABLE", "", nil
		}
	}
	return &transactionID, "", "", nil
}

func (r *DBRepository) resolveAirwallexRelationshipTarget(
	ctx context.Context,
	providerAccountKey string,
	providerTransactionID string,
) (airwallexRelationshipTarget, bool, bool, error) {
	rows, err := r.db.QueryContext(ctx, selectAirwallexRelationshipTargetSQL, providerAccountKey, providerTransactionID)
	if err != nil {
		return airwallexRelationshipTarget{}, false, false, fmt.Errorf("resolve Airwallex relationship target: %w", err)
	}
	defer rows.Close()
	targets := make([]airwallexRelationshipTarget, 0, 2)
	for rows.Next() {
		var item airwallexRelationshipTarget
		var amount string
		var fromID, toID sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&item.Key,
			&item.Kind,
			&item.Currency,
			&amount,
			&item.Direction,
			&fromID,
			&toID,
		); err != nil {
			return airwallexRelationshipTarget{}, false, false, err
		}
		item.Amount, err = decimal.NewFromString(amount)
		if err != nil {
			return airwallexRelationshipTarget{}, false, false, fmt.Errorf("parse Airwallex relationship target amount: %w", err)
		}
		if fromID.Valid {
			value := fromID.Int64
			item.FromID = &value
		}
		if toID.Valid {
			value := toID.Int64
			item.ToID = &value
		}
		targets = append(targets, item)
	}
	if err := rows.Err(); err != nil {
		return airwallexRelationshipTarget{}, false, false, err
	}
	if len(targets) == 0 {
		return airwallexRelationshipTarget{}, false, false, nil
	}
	if len(targets) != 1 {
		return airwallexRelationshipTarget{}, false, true, nil
	}
	return targets[0], true, false, nil
}

func validateAirwallexReversalSemantics(
	proposal TransactionUpsertInput,
	target airwallexRelationshipTarget,
) string {
	if proposal.Currency != target.Currency ||
		!proposal.Amount.IsPositive() ||
		proposal.Amount.GreaterThan(target.Amount) {
		return "REVERSAL_VALUE_CONFLICT"
	}
	switch target.Direction {
	case DirectionOutflow:
		if proposal.Direction != DirectionInflow ||
			target.FromID == nil ||
			proposal.ToCompanyFundAccountID == nil ||
			*target.FromID != *proposal.ToCompanyFundAccountID {
			return "REVERSAL_DIRECTION_ACCOUNT_CONFLICT"
		}
	case DirectionInflow:
		if proposal.Direction != DirectionOutflow ||
			target.ToID == nil ||
			proposal.FromCompanyFundAccountID == nil ||
			*target.ToID != *proposal.FromCompanyFundAccountID {
			return "REVERSAL_DIRECTION_ACCOUNT_CONFLICT"
		}
	case DirectionInternalTransfer:
		if proposal.Direction != DirectionInternalTransfer ||
			target.FromID == nil ||
			target.ToID == nil ||
			proposal.FromCompanyFundAccountID == nil ||
			proposal.ToCompanyFundAccountID == nil ||
			*target.FromID != *proposal.ToCompanyFundAccountID ||
			*target.ToID != *proposal.FromCompanyFundAccountID {
			return "REVERSAL_DIRECTION_ACCOUNT_CONFLICT"
		}
	default:
		return "REVERSAL_DIRECTION_ACCOUNT_CONFLICT"
	}
	return ""
}

func (processor *AirwallexLedgerTaskProcessor) processConversion(
	ctx context.Context,
	lease CompanyFundLedgerTaskLease,
) (*int64, string, string, error) {
	peer, found, conflict, err := processor.repository.loadAirwallexConversionPeer(ctx, lease)
	if err != nil {
		return nil, "", "", err
	}
	if conflict {
		return nil, "", "CONVERSION_GROUP_CONFLICT", nil
	}
	if !found {
		return nil, "CONVERSION_PEER_WAITING", "", nil
	}
	leaseFact, err := processor.repository.loadAirwallexConversionFact(
		ctx,
		lease.ProviderTransactionFactID,
		lease.ProviderAccountKey,
	)
	if err != nil {
		return nil, "", "", err
	}
	peerFact, err := processor.repository.loadAirwallexConversionFact(
		ctx,
		peer.ProviderTransactionFactID,
		lease.ProviderAccountKey,
	)
	if err != nil {
		return nil, "", "", err
	}
	if conflictCode := validateAirwallexConversionPairEvidence(lease, peer, leaseFact, peerFact); conflictCode != "" {
		return nil, "", conflictCode, nil
	}
	if lease.Proposal.ConversionLeg == peer.Proposal.ConversionLeg ||
		!lease.Proposal.ConversionLeg.Valid() || !peer.Proposal.ConversionLeg.Valid() {
		return nil, "", "CONVERSION_LEG_CONFLICT", nil
	}
	first := lease.Proposal
	first.ProviderTransactionFactID = &lease.ProviderTransactionFactID
	second := peer.Proposal
	second.ProviderTransactionFactID = &peer.ProviderTransactionFactID
	for _, proposal := range []*TransactionUpsertInput{&first, &second} {
		proposal.RelationshipReferenceType = RelationshipReferenceSourceIDConversion
		proposal.RelationshipReferenceKey = lease.RelationshipReferenceKey
		proposal.RelationshipGroupKey = lease.RelationshipGroupKey
		proposal.ConversionGroupKey = lease.RelationshipGroupKey
		proposal.ConversionGroupState = ConversionGroupIncomplete
	}
	proposals := []TransactionUpsertInput{first, second}
	if proposals[1].MovementKey < proposals[0].MovementKey {
		proposals[0], proposals[1] = proposals[1], proposals[0]
	}
	supplements := make([]normalizedTransactionProviderSupplement, len(proposals))
	for index := range proposals {
		if err := proposals[index].validate(); err != nil {
			return nil, "", "CONVERSION_PROPOSAL_INVALID", nil
		}
		supplement, err := normalizeTransactionProviderSupplement(
			proposals[index].ProviderDisplay,
			proposals[index].AutomaticRisk,
		)
		if err != nil {
			return nil, "", "CONVERSION_PROPOSAL_INVALID", nil
		}
		supplements[index] = supplement
	}
	tx, err := processor.repository.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", "", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var leaseTransactionID int64
	for index, proposal := range proposals {
		result, err := processor.repository.upsertCompanyFundTransactionTx(
			ctx,
			tx,
			proposal,
			supplements[index],
		)
		if err != nil {
			var quarantine *TransactionQuarantineError
			if errors.As(err, &quarantine) {
				return nil, "", "CONVERSION_GROUP_CONFLICT", nil
			}
			return nil, "", "", err
		}
		if proposal.MovementKey == lease.Proposal.MovementKey {
			leaseTransactionID = result.ID
		}
	}
	completed, err := processor.repository.completeAirwallexConversionGroupTx(
		ctx,
		tx,
		lease.ProviderAccountKey,
		lease.RelationshipGroupKey,
	)
	if err != nil {
		return nil, "", "", err
	}
	if !completed || leaseTransactionID <= 0 {
		return nil, "", "CONVERSION_GROUP_INVALID", nil
	}
	if err := tx.Commit(); err != nil {
		return nil, "", "", fmt.Errorf("commit Airwallex conversion group: %w", err)
	}
	committed = true
	return &leaseTransactionID, "", "", nil
}

func (r *DBRepository) loadAirwallexConversionPeer(
	ctx context.Context,
	lease CompanyFundLedgerTaskLease,
) (CompanyFundLedgerTaskLease, bool, bool, error) {
	rows, err := r.db.QueryContext(ctx, selectAirwallexConversionPeerTaskSQL,
		lease.ProviderAccountKey, lease.RelationshipGroupKey, lease.ID,
	)
	if err != nil {
		return CompanyFundLedgerTaskLease{}, false, false, err
	}
	defer rows.Close()
	peers := make([]CompanyFundLedgerTaskLease, 0, 2)
	for rows.Next() {
		var peer CompanyFundLedgerTaskLease
		var payload, digest, relationshipType string
		if err := rows.Scan(&peer.ID, &peer.ProviderTransactionFactID,
			&peer.SubjectProviderTransactionID,
			&relationshipType,
			&peer.RelationshipReferenceKey,
			&peer.RelationshipGroupKey,
			&peer.EvidenceReference,
			&payload,
			&digest,
		); err != nil {
			return CompanyFundLedgerTaskLease{}, false, false, err
		}
		peer.Channel = ChannelAirwallex
		peer.ProviderAccountKey = lease.ProviderAccountKey
		peer.RelationshipReferenceType = RelationshipReferenceType(relationshipType)
		if payloadSHA256Hex([]byte(payload)) != digest {
			return CompanyFundLedgerTaskLease{}, false, true, nil
		}
		peer.Proposal, err = decodeCompanyFundLedgerTaskPayload([]byte(payload))
		if err != nil {
			return CompanyFundLedgerTaskLease{}, false, true, nil
		}
		peers = append(peers, peer)
	}
	if len(peers) == 0 {
		return CompanyFundLedgerTaskLease{}, false, false, nil
	}
	if len(peers) != 1 {
		return CompanyFundLedgerTaskLease{}, false, true, nil
	}
	return peers[0], true, false, nil
}

func (r *DBRepository) loadAirwallexConversionFact(
	ctx context.Context,
	factID int64,
	providerAccountKey string,
) (ProviderTransactionFact, error) {
	fact, err := scanProviderTransactionFact(
		r.db.QueryRowContext(ctx, selectAirwallexConversionFactByIDSQL, factID, providerAccountKey),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProviderTransactionFact{}, fmt.Errorf("Airwallex conversion fact %d is unavailable", factID)
		}
		return ProviderTransactionFact{}, fmt.Errorf("load Airwallex conversion fact %d: %w", factID, err)
	}
	return fact, nil
}

func validateAirwallexConversionPairEvidence(
	lease CompanyFundLedgerTaskLease,
	peer CompanyFundLedgerTaskLease,
	leaseFact ProviderTransactionFact,
	peerFact ProviderTransactionFact,
) string {
	if lease.RelationshipReferenceType != RelationshipReferenceSourceIDConversion ||
		peer.RelationshipReferenceType != lease.RelationshipReferenceType ||
		peer.RelationshipReferenceKey != lease.RelationshipReferenceKey ||
		peer.RelationshipGroupKey != lease.RelationshipGroupKey ||
		peer.EvidenceReference != lease.EvidenceReference {
		return "CONVERSION_RELATIONSHIP_EVIDENCE_CONFLICT"
	}
	if lease.Proposal.Direction != DirectionInternalTransfer ||
		peer.Proposal.Direction != DirectionInternalTransfer ||
		lease.Proposal.FromCompanyFundAccountID == nil ||
		lease.Proposal.ToCompanyFundAccountID == nil ||
		peer.Proposal.FromCompanyFundAccountID == nil ||
		peer.Proposal.ToCompanyFundAccountID == nil ||
		*lease.Proposal.FromCompanyFundAccountID != *lease.Proposal.ToCompanyFundAccountID ||
		*peer.Proposal.FromCompanyFundAccountID != *peer.Proposal.ToCompanyFundAccountID ||
		*lease.Proposal.FromCompanyFundAccountID != *peer.Proposal.FromCompanyFundAccountID {
		return "CONVERSION_ACCOUNT_DIRECTION_CONFLICT"
	}
	if leaseFact.ProviderTransactionID != lease.SubjectProviderTransactionID ||
		peerFact.ProviderTransactionID != peer.SubjectProviderTransactionID ||
		leaseFact.ProviderSourceReference == "" ||
		leaseFact.ProviderSourceReference != lease.RelationshipGroupKey ||
		peerFact.ProviderSourceReference != leaseFact.ProviderSourceReference ||
		leaseFact.ConversionFromCurrency == "" ||
		leaseFact.ConversionToCurrency == "" ||
		leaseFact.ConversionFromCurrency == leaseFact.ConversionToCurrency ||
		peerFact.ConversionFromCurrency != leaseFact.ConversionFromCurrency ||
		peerFact.ConversionToCurrency != leaseFact.ConversionToCurrency ||
		leaseFact.ConversionRate == nil ||
		peerFact.ConversionRate == nil ||
		!leaseFact.ConversionRate.IsPositive() ||
		!leaseFact.ConversionRate.Equal(*peerFact.ConversionRate) {
		return "CONVERSION_PAIR_FACT_CONFLICT"
	}
	if leaseFact.ProviderAmount == nil ||
		peerFact.ProviderAmount == nil ||
		leaseFact.ProviderCurrency != lease.Proposal.Currency ||
		peerFact.ProviderCurrency != peer.Proposal.Currency ||
		!leaseFact.ProviderAmount.Equal(lease.Proposal.Amount) ||
		!peerFact.ProviderAmount.Equal(peer.Proposal.Amount) {
		return "CONVERSION_LEG_VALUE_CONFLICT"
	}
	for _, item := range []struct {
		proposal TransactionUpsertInput
		fact     ProviderTransactionFact
	}{
		{proposal: lease.Proposal, fact: leaseFact},
		{proposal: peer.Proposal, fact: peerFact},
	} {
		expectedCurrency := item.fact.ConversionFromCurrency
		if item.proposal.ConversionLeg == ConversionLegBuy {
			expectedCurrency = item.fact.ConversionToCurrency
		}
		if !item.proposal.ConversionLeg.Valid() || item.proposal.Currency != expectedCurrency {
			return "CONVERSION_LEG_CURRENCY_CONFLICT"
		}
	}
	return ""
}

func (r *DBRepository) completeAirwallexConversionGroupTx(
	ctx context.Context,
	tx *sql.Tx,
	providerAccountKey string,
	groupKey string,
) (bool, error) {
	var count int
	if err := tx.QueryRowContext(ctx, completeAirwallexConversionGroupSQL,
		providerAccountKey, groupKey,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("complete Airwallex conversion group: %w", err)
	}
	return count == 2, nil
}
