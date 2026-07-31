package companyfund

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/shopspring/decimal"
)

const airwallexPhase2PostgresGate = "RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION"

func TestAirwallexPayoutCounterpartyPostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL Airwallex Phase 2 coverage")
	}
	db := newAirwallexPhase2PostgresFixture(t, requiredAirwallexPhase2DatabaseURL(t))
	ctx := context.Background()
	source := loadAirwallexPhase2Source(t, db)
	registry, err := buildAccountRegistrySnapshot([]CompanyFundAccount{{
		ID:                 source.accountID,
		Channel:            AccountChannelAirwallex,
		ProviderAccountKey: source.accountKey,
		CompanyEntity:      "Monera Ltd",
		FundAccountName:    "Treasury",
		AccountType:        "BANK",
		AccountName:        "USD account",
		Enabled:            true,
	}}, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	config := testAirwallexRuntimePayoutConfig()
	config.Rules[0].ProviderAccountKey = source.accountKey
	feeAmount := decimal.RequireFromString("15.58")
	feeCurrency := "USD"
	transferDetailsClient := &airwallexTransferDetailsClientStub{
		beneficiary: AirwallexTransferBeneficiary{
			AddressOrAccount: "1234567890",
			Name:             "Ada Recipient",
		},
		fee: ProviderTransactionFeeInput{
			Amount:   &feeAmount,
			Currency: &feeCurrency,
			DetailsJSON: []byte(
				`{"amountBeneficiaryReceives":984.42,"amountPayerPays":1000,"enrichmentSource":"AIRWALLEX_TRANSFER_API","feePaidBy":"BENEFICIARY","swiftChargeOption":"SHARED"}`,
			),
		},
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundleWithTransferDetailsClient(
		config,
		&airwallexProviderEventRegistryStub{snapshot: registry},
		&airwallexTransferDetailsFactoryStub{client: transferDetailsClient},
	)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"id":"ft_payout_counterparty_1","amount":-19.75,"fee":0,"net":-19.75,"created_at":"2026-07-10T03:00:00Z","currency":"USD","source_id":"transfer_1","source_type":"PAYOUT","status":"SETTLED","transaction_type":"PAYOUT"}`)
	digest := payloadSHA256Hex(payload)
	eventInput := ProviderEventInput{
		Channel:                       ChannelAirwallex,
		ProviderEventID:               "airwallex-payout-counterparty-integration",
		EventType:                     AirwallexFinancialTransactionSnapshotEventType,
		ProviderEventVersion:          airwallexTestAPIVersion,
		ProviderAccountKey:            source.accountKey,
		SourceKind:                    ProviderEventSourceOwnedEncryptedPayload,
		SourcePayloadDigest:           digest,
		OwnedPayloadCiphertext:        []byte("integration-ciphertext"),
		OwnedPayloadDigest:            digest,
		OwnedPayloadKeyVersion:        "integration-v1",
		OwnedPayloadRetentionDuration: time.Hour,
	}
	repository := NewDBRepository(db)
	inserted, err := repository.InsertProviderEvent(ctx, eventInput)
	if err != nil || !inserted.Inserted {
		t.Fatalf("insert payout provider event = %#v, %v", inserted, err)
	}
	worker := newProviderEventWorkerForTest(
		t,
		repository,
		&providerEventPayloadReaderStub{payload: payload},
		map[TransactionSource]ProviderEventNormalizer{ChannelAirwallex: bundle.ProviderEvents},
		time.Now(),
	)
	result, err := worker.ProcessNext(ctx)
	if err != nil || result.Outcome != ProviderEventFinalizeProcessed || result.MovementCount != 1 {
		t.Fatalf("process payout provider event = %#v, %v", result, err)
	}
	assertAirwallexPayoutDetails(t, db, "ft_payout_counterparty_1", "1234567890", "Ada Recipient", 1)

	if _, err := db.ExecContext(ctx, `
UPDATE company_fund_transactions
SET provider_reported_fee_amount = NULL,
	provider_reported_fee_currency = NULL,
	fee_details = '{"legacy":"keep"}'::jsonb
WHERE provider_transaction_id = 'ft_payout_counterparty_1'`); err != nil {
		t.Fatal(err)
	}
	requeued, err := repository.RequeueAirwallexPayoutCounterpartyBackfill(ctx, 10)
	if err != nil || requeued != 1 {
		t.Fatalf("RequeueAirwallexPayoutCounterpartyBackfill(fee) = %d, %v", requeued, err)
	}
	result, err = worker.ProcessNext(ctx)
	if err != nil || result.Outcome != ProviderEventFinalizeProcessed || result.MovementCount != 1 {
		t.Fatalf("process payout fee backfill = %#v, %v", result, err)
	}
	assertAirwallexPayoutDetails(t, db, "ft_payout_counterparty_1", "1234567890", "Ada Recipient", 1)
	if candidatesRemain, err := repository.HasAirwallexPayoutCounterpartyBackfillCandidates(ctx); err != nil || candidatesRemain {
		t.Fatalf("HasAirwallexPayoutCounterpartyBackfillCandidates() = %t, %v, want drained", candidatesRemain, err)
	}

	if _, err := db.ExecContext(ctx, `
UPDATE company_fund_transactions
SET to_address_or_account = NULL, payee_name = NULL
WHERE provider_transaction_id = 'ft_payout_counterparty_1'`); err != nil {
		t.Fatal(err)
	}
	requeued, err = repository.RequeueAirwallexPayoutCounterpartyBackfill(ctx, 10)
	if err != nil || requeued != 1 {
		t.Fatalf("RequeueAirwallexPayoutCounterpartyBackfill() = %d, %v", requeued, err)
	}
	result, err = worker.ProcessNext(ctx)
	if err != nil || result.Outcome != ProviderEventFinalizeProcessed || result.MovementCount != 1 {
		t.Fatalf("process payout backfill = %#v, %v", result, err)
	}
	assertAirwallexPayoutDetails(t, db, "ft_payout_counterparty_1", "1234567890", "Ada Recipient", 1)

	replayed, err := repository.InsertProviderEvent(ctx, eventInput)
	if err != nil || replayed.Inserted || replayed.ID != inserted.ID {
		t.Fatalf("replay payout provider event = %#v, %v", replayed, err)
	}
	assertAirwallexPayoutDetails(t, db, "ft_payout_counterparty_1", "1234567890", "Ada Recipient", 1)

	transferDetailsClient.beneficiary = AirwallexTransferBeneficiary{}
	if _, err := db.ExecContext(ctx, `
UPDATE company_fund_provider_events
SET event_state = 'PENDING', processed_at = NULL, updated_at = NOW()
WHERE id = $1`, inserted.ID); err != nil {
		t.Fatal(err)
	}
	result, err = worker.ProcessNext(ctx)
	if err != nil || result.Outcome != ProviderEventFinalizeFailed {
		t.Fatalf("empty beneficiary replay = %#v, %v", result, err)
	}
	assertAirwallexPayoutDetails(t, db, "ft_payout_counterparty_1", "1234567890", "Ada Recipient", 1)
}

// TestAirwallexPhase2FeePostgresIntegration proves the highest durable seam:
// provider fact + waiting task -> linked movement + automatic finance category.
// It uses cloned tables in a unique schema and never changes public data.
func TestAirwallexPhase2FeePostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL Airwallex Phase 2 coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when Airwallex Phase 2 integration coverage is enabled")
	}
	db := newAirwallexPhase2PostgresFixture(t, databaseURL)
	ctx := context.Background()

	var (
		accountID        int64
		accountKey       string
		parentProviderID string
		eventID          int64
		providerEventID  string
		payloadDigest    string
	)
	if err := db.QueryRowContext(ctx, `
SELECT COALESCE(transaction.from_company_fund_account_id, transaction.to_company_fund_account_id),
       transaction.provider_account_key,
       transaction.provider_transaction_id,
       event.id,
       event.provider_event_id,
       event.source_payload_digest
FROM company_fund_transactions transaction
JOIN company_fund_provider_events event
  ON event.id = transaction.latest_provider_event_id
WHERE transaction.channel = 'AIRWALLEX'
  AND transaction.provider_account_key IS NOT NULL
  AND transaction.provider_transaction_id IS NOT NULL
  AND (transaction.from_company_fund_account_id IS NOT NULL OR transaction.to_company_fund_account_id IS NOT NULL)
ORDER BY transaction.id
LIMIT 1`).Scan(
		&accountID,
		&accountKey,
		&parentProviderID,
		&eventID,
		&providerEventID,
		&payloadDigest,
	); err != nil {
		t.Fatalf("load Airwallex parent fixture: %v", err)
	}

	feeProviderID := fmt.Sprintf("phase2-fee-%d", time.Now().UnixNano())
	factIdentity := "phase2-fact:" + feeProviderID
	var factID int64
	if err := db.QueryRowContext(ctx, `
INSERT INTO company_fund_provider_transaction_facts (
	channel, provider_account_key, provider_transaction_id,
	provider_source_reference, fact_identity_key, fact_version,
	source_provider_event_id, source_payload_digest,
	provider_amount, provider_currency, value_scope, allocation_state,
	provider_extras
) VALUES (
	'AIRWALLEX', $1, $2, $3, $4, 1, $5, $6,
	0.75, 'USD', 'DIRECT_ITEM', 'NOT_APPLICABLE', '{}'::jsonb
)
RETURNING id`, accountKey, feeProviderID, parentProviderID, factIdentity, eventID, payloadDigest).Scan(&factID); err != nil {
		t.Fatalf("insert fee fact fixture: %v", err)
	}
	proposal := validProviderEventWorkerMovement("phase2:" + feeProviderID)
	proposal.ProviderAccountKey = accountKey
	proposal.ProviderTransactionID = feeProviderID
	proposal.ProviderEventID = providerEventID
	proposal.ProviderMovementID = feeProviderID
	proposal.ProviderTransactionFactID = nil
	proposal.MovementKind = MovementKindFee
	proposal.Direction = DirectionOutflow
	proposal.FromCompanyFundAccountID = &accountID
	proposal.ToCompanyFundAccountID = nil
	proposal.Amount = decimal.RequireFromString("0.75")
	proposal.FirstSeenSource = TransactionSeenSourceReconciliation
	proposal.LatestProviderEventID = &eventID
	proposal.RawSnapshotDigest = payloadDigest
	proposal.Provider.Metadata.Source = ProviderSourceReconciliation

	repository := NewDBRepository(db)
	enqueued, err := repository.EnqueueCompanyFundLedgerTask(ctx, CompanyFundLedgerTaskInput{
		Channel:                      ChannelAirwallex,
		ProviderAccountKey:           accountKey,
		Kind:                         LedgerTaskKindFeeRelationship,
		ProviderTransactionFactID:    factID,
		SubjectProviderTransactionID: feeProviderID,
		RelationshipReferenceType:    RelationshipReferenceSourceIDExactParent,
		RelationshipReferenceKey:     parentProviderID,
		EvidenceReference:            "postgres-integration-exact-parent",
		Proposal:                     proposal,
		RelationshipSLA:              time.Hour,
	})
	if err != nil || !enqueued.Inserted {
		t.Fatalf("enqueue fee relationship = %#v, %v", enqueued, err)
	}
	processor, err := NewAirwallexLedgerTaskProcessor(repository, AirwallexLedgerTaskProcessorConfig{
		Owner:         "phase2-integration",
		LeaseDuration: time.Minute,
		RetryDelay:    time.Second,
		FeeClassification: AirwallexFeeClassificationPolicy{
			Level1Code:    "PHASE2_OPERATING_EXPENSE",
			Level2Code:    "PHASE2_FEE",
			PolicyVersion: "phase2-fee-v1",
		},
		ReversalPolicyVersion: "phase2-reversal-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := processor.ProcessNext(ctx)
	if err != nil || processed.Outcome != LedgerTaskProcessRetrying {
		t.Fatalf("ProcessNext() without categories = %#v, %v", processed, err)
	}
	var (
		pendingTransactionID int64
		pendingSource        string
	)
	if err := db.QueryRowContext(ctx, `
SELECT id, classification_source
FROM company_fund_transactions
WHERE provider_transaction_id = $1`, feeProviderID).Scan(
		&pendingTransactionID,
		&pendingSource,
	); err != nil {
		t.Fatalf("read durable fee pending classification: %v", err)
	}
	if pendingTransactionID <= 0 || pendingSource != "UNCLASSIFIED" {
		t.Fatalf("durable fee pending classification id=%d source=%s", pendingTransactionID, pendingSource)
	}

	if _, err := db.ExecContext(ctx, `
INSERT INTO finance_categories (level, code, name, is_enabled)
VALUES (1, 'PHASE2_OPERATING_EXPENSE', 'Operating expense', true)`); err != nil {
		t.Fatalf("insert level-one category fixture: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO finance_categories (level, parent_id, code, name, is_enabled)
SELECT 2, id, 'PHASE2_FEE', 'Fee', true
FROM finance_categories WHERE code = 'PHASE2_OPERATING_EXPENSE'`); err != nil {
		t.Fatalf("insert level-two category fixture: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
UPDATE company_fund_ledger_tasks
SET next_attempt_at = clock_timestamp()
WHERE id = $1`, processed.TaskID); err != nil {
		t.Fatalf("make fee classification retryable: %v", err)
	}
	processed, err = processor.ProcessNext(ctx)
	if err != nil || processed.Outcome != LedgerTaskProcessCompleted {
		t.Fatalf("ProcessNext() after category recovery = %#v, %v", processed, err)
	}

	var (
		parentID       sql.NullInt64
		source         string
		policyVersion  string
		operating      bool
		summaryInclude bool
	)
	if err := db.QueryRowContext(ctx, `
SELECT parent_transaction_id,
       classification_source,
       classification_policy_version,
       is_operating_income_expense,
       summary_inclusion_override
FROM company_fund_transactions
WHERE provider_transaction_id = $1`, feeProviderID).Scan(
		&parentID,
		&source,
		&policyVersion,
		&operating,
		&summaryInclude,
	); err != nil {
		t.Fatalf("read completed fee movement: %v", err)
	}
	if !parentID.Valid || source != "AUTO_RULE" || policyVersion != "phase2-fee-v1" ||
		!operating || !summaryInclude {
		t.Fatalf("completed fee movement parent=%v source=%s policy=%s operating=%t summary=%t",
			parentID, source, policyVersion, operating, summaryInclude)
	}
}

func TestAirwallexPhase2FeeBackfillPostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL Airwallex Phase 2 coverage")
	}
	db := newAirwallexPhase2PostgresFixture(t, requiredAirwallexPhase2DatabaseURL(t))
	ctx := context.Background()
	source := loadAirwallexPhase2Source(t, db)
	repository := NewDBRepository(db)
	if _, err := db.ExecContext(ctx, `
INSERT INTO finance_categories (level, code, name, is_enabled)
VALUES (1, 'PHASE2_OPERATING_EXPENSE', 'Operating expense', true)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO finance_categories (level, parent_id, code, name, is_enabled)
SELECT 2, id, 'PHASE2_FEE', 'Fee', true
FROM finance_categories
WHERE code = 'PHASE2_OPERATING_EXPENSE'`); err != nil {
		t.Fatal(err)
	}

	providerIDs := []string{
		fmt.Sprintf("phase2-backfill-auto-%d", time.Now().UnixNano()),
		fmt.Sprintf("phase2-backfill-manual-%d", time.Now().UnixNano()),
	}
	transactionIDs := make([]int64, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		factID := insertAirwallexPhase2Fact(t, db, source, providerID, "fee-group", "0.01", "USD")
		proposal := airwallexPhase2Proposal(source, "phase2:"+providerID, providerID, "0.01", "USD")
		proposal.ProviderTransactionFactID = &factID
		proposal.MovementKind = MovementKindFee
		proposal.Direction = DirectionOutflow
		proposal.FromCompanyFundAccountID = &source.accountID
		proposal.RelationshipReferenceType = RelationshipReferenceSourceIDGroupOnly
		proposal.RelationshipReferenceKey = "fee-group"
		proposal.RelationshipGroupKey = "fee-group"
		included := false
		proposal.AutomaticRisk.AutoExcludedFromSummary = &included
		inserted, err := repository.UpsertCompanyFundTransaction(ctx, proposal)
		if err != nil {
			t.Fatalf("insert historical fee %s: %v", providerID, err)
		}
		transactionIDs = append(transactionIDs, inserted.ID)
	}
	if _, err := repository.UpdateFinanceTransactionClassification(ctx, FinanceClassificationUpdate{
		TransactionID: transactionIDs[1],
		UpdatedBy:     "phase2-postgres-integration-manual-clear",
	}); err != nil {
		t.Fatalf("mark historical fee manual clear: %v", err)
	}

	processorV1 := newAirwallexPhase2Processor(t, repository, "phase2-backfill-v1")
	if maintained, err := processorV1.Maintain(ctx); err != nil || !maintained {
		t.Fatalf("Maintain(v1) = %t, %v", maintained, err)
	}
	drainAirwallexPhase2Tasks(t, ctx, processorV1)
	assertAirwallexPhase2Classification(t, db, transactionIDs[0], "AUTO_RULE", "phase2-fee-v1")
	assertAirwallexPhase2Classification(t, db, transactionIDs[1], "MANUAL", "")

	processorV2 := newAirwallexPhase2Processor(
		t,
		repository,
		"phase2-backfill-v2",
		"phase2-fee-v2",
	)
	if maintained, err := processorV2.Maintain(ctx); err != nil || !maintained {
		t.Fatalf("Maintain(v2) = %t, %v", maintained, err)
	}
	drainAirwallexPhase2Tasks(t, ctx, processorV2)
	assertAirwallexPhase2Classification(t, db, transactionIDs[0], "AUTO_RULE", "phase2-fee-v2")
	assertAirwallexPhase2Classification(t, db, transactionIDs[1], "MANUAL", "")

	var manualTasks int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM company_fund_ledger_tasks
WHERE subject_transaction_id = $1
  AND task_kind = 'FEE_CLASSIFICATION'`, transactionIDs[1]).Scan(&manualTasks); err != nil {
		t.Fatal(err)
	}
	if manualTasks != 0 {
		t.Fatalf("manual historical fee classification task count = %d, want 0", manualTasks)
	}
}

// TestAirwallexPhase2ConversionProviderEventPostgresIntegration proves the
// complete local production seam: durable provider event -> normalized facts
// and tasks -> atomically visible conversion pair.
func TestAirwallexPhase2ConversionProviderEventPostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL Airwallex Phase 2 coverage")
	}
	db := newAirwallexPhase2PostgresFixture(t, requiredAirwallexPhase2DatabaseURL(t))
	ctx := context.Background()
	source := loadAirwallexPhase2Source(t, db)
	repository := NewDBRepository(db)
	repository.SetSafeheronProviderClaimMode(SafeheronProviderClaimDisabled)
	if _, err := db.ExecContext(ctx, `
UPDATE company_fund_provider_events
SET event_state = 'PROCESSED', lease_owner = NULL, lease_expires_at = NULL
WHERE event_state IN ('PENDING', 'LEASED', 'FAILED')`); err != nil {
		t.Fatalf("isolate provider-event queue: %v", err)
	}
	snapshot, err := buildAccountRegistrySnapshot([]CompanyFundAccount{{
		ID:                 source.accountID,
		Channel:            AccountChannelAirwallex,
		ProviderAccountKey: source.accountKey,
		Enabled:            true,
	}}, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	rules := make([]AirwallexFinancialTransactionsRuntimeRule, 0, 2)
	for _, leg := range []struct {
		transactionType string
		currency        string
		conversionLeg   ConversionLeg
		expectedSign    AirwallexFinancialValueSign
	}{
		{"CONVERSION_SELL", "USD", ConversionLegSell, AirwallexFinancialValueSignNegative},
		{"CONVERSION_BUY", "SGD", ConversionLegBuy, AirwallexFinancialValueSignPositive},
	} {
		rules = append(rules, AirwallexFinancialTransactionsRuntimeRule{
			EvidenceReference:  "phase2-provider-event-conversion",
			ProviderAccountKey: source.accountKey,
			Currency:           leg.currency,
			Status:             "SETTLED",
			Classification: AirwallexFinancialTransactionClassification{
				TransactionType: leg.transactionType,
				SourceType:      "CONVERSION",
				Action:          AirwallexFinancialTransactionActionApply,
				MovementKind:    MovementKindConversion,
				Direction:       DirectionInternalTransfer,
				TransferMode:    TransferModeSingle,
				AmountField:     AirwallexFinancialAmountFieldAmount,
				ExpectedSign:    leg.expectedSign,
				OccurredAtField: AirwallexFinancialOccurredAtCreated,
				ClientRateUse:   AirwallexFinancialClientRateUseConversionRate,
			},
			Relationship: AirwallexRuntimeRelationshipRule{
				Strategy:          AirwallexRuntimeRelationshipSourceConversion,
				EvidenceReference: "phase2-provider-event-conversion",
				ConversionLeg:     leg.conversionLeg,
				FromCurrency:      "USD",
				ToCurrency:        "SGD",
				SLADuration:       time.Hour,
			},
		})
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(
		AirwallexFinancialTransactionsRuntimeConfig{
			Enabled:        true,
			APIVersion:     airwallexTestAPIVersion,
			SchemaVersion:  "schema-v1",
			EventVersion:   "event-v1",
			MappingVersion: "phase2-provider-event-v1",
			FactVersion:    1,
			Rules:          rules,
		},
		&airwallexProviderEventRegistryStub{snapshot: snapshot},
	)
	if err != nil {
		t.Fatal(err)
	}
	groupKey := fmt.Sprintf("phase2-provider-event-conversion-%d", time.Now().UnixNano())
	payloads := [][]byte{
		[]byte(`{"id":"` + groupKey + `-sell","amount":-10,"fee":0,"net":-10,"client_rate":1.35,"created_at":"2026-07-10T01:02:03Z","currency":"USD","currency_pair":"USDSGD","source_id":"` + groupKey + `","source_type":"CONVERSION","status":"SETTLED","transaction_type":"CONVERSION_SELL"}`),
		[]byte(`{"id":"` + groupKey + `-buy","amount":13.5,"fee":0,"net":13.5,"client_rate":1.35,"created_at":"2026-07-10T01:02:04Z","currency":"SGD","currency_pair":"USDSGD","source_id":"` + groupKey + `","source_type":"CONVERSION","status":"SETTLED","transaction_type":"CONVERSION_BUY"}`),
	}
	for index, payload := range payloads {
		digest := payloadSHA256Hex(payload)
		eventID := fmt.Sprintf("phase2-provider-event-%d-%s", index, groupKey)
		input := ProviderEventInput{
			Channel:                       ChannelAirwallex,
			ProviderEventID:               eventID,
			EventType:                     AirwallexFinancialTransactionSnapshotEventType,
			ProviderEventVersion:          airwallexTestAPIVersion,
			ProviderAccountKey:            source.accountKey,
			SourceKind:                    ProviderEventSourceOwnedEncryptedPayload,
			SourcePayloadDigest:           digest,
			OwnedPayloadCiphertext:        []byte("integration-ciphertext"),
			OwnedPayloadDigest:            digest,
			OwnedPayloadKeyVersion:        "integration-v1",
			OwnedPayloadRetentionDuration: time.Hour,
		}
		inserted, err := repository.InsertProviderEvent(ctx, input)
		if err != nil || !inserted.Inserted {
			t.Fatalf("insert provider event %d = %#v, %v", index, inserted, err)
		}
		worker := newProviderEventWorkerForTest(
			t,
			repository,
			&providerEventPayloadReaderStub{payload: payload},
			map[TransactionSource]ProviderEventNormalizer{ChannelAirwallex: bundle.ProviderEvents},
			time.Now(),
		)
		result, err := worker.ProcessNext(ctx)
		if err != nil || result.Outcome != ProviderEventFinalizeProcessed ||
			result.FactCount != 1 || result.TaskCount != 1 || result.MovementCount != 0 {
			t.Fatalf("process provider event %d = %#v, %v", index, result, err)
		}
		replayed, err := repository.InsertProviderEvent(ctx, input)
		if err != nil || replayed.Inserted || replayed.ID != inserted.ID {
			t.Fatalf("replay provider event %d = %#v, %v", index, replayed, err)
		}
	}
	processor := newAirwallexPhase2Processor(t, repository, "phase2-provider-event-conversion")
	result, err := processor.ProcessNext(ctx)
	if err != nil || result.Outcome != LedgerTaskProcessCompleted {
		t.Fatalf("process provider-event conversion = %#v, %v", result, err)
	}
	var facts, tasks, movements int
	if err := db.QueryRowContext(ctx, `
SELECT
	(SELECT COUNT(*) FROM company_fund_provider_transaction_facts WHERE provider_source_reference = $1),
	(SELECT COUNT(*) FROM company_fund_ledger_tasks WHERE relationship_group_key = $1),
	(SELECT COUNT(*) FROM company_fund_transactions
	 WHERE conversion_group_key = $1 AND conversion_group_status = 'COMPLETE')`,
		groupKey,
	).Scan(&facts, &tasks, &movements); err != nil {
		t.Fatal(err)
	}
	if facts != 2 || tasks != 2 || movements != 2 {
		t.Fatalf("provider-event conversion facts=%d tasks=%d movements=%d", facts, tasks, movements)
	}
}

// TestAirwallexPhase2ConversionPostgresIntegration proves that a conversion
// remains invisible with only one leg, then becomes visible as exactly one
// SELL and one BUY movement on the same multi-currency Airwallex account.
func TestAirwallexPhase2ConversionPostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL Airwallex Phase 2 coverage")
	}
	type conversionLegFixture struct {
		providerID string
		currency   string
		amount     string
		leg        ConversionLeg
	}
	for _, testCase := range []struct {
		name  string
		first conversionLegFixture
		last  conversionLegFixture
	}{
		{
			name:  "sell first",
			first: conversionLegFixture{currency: "USD", amount: "10", leg: ConversionLegSell},
			last:  conversionLegFixture{currency: "SGD", amount: "13.5", leg: ConversionLegBuy},
		},
		{
			name:  "buy first",
			first: conversionLegFixture{currency: "SGD", amount: "13.5", leg: ConversionLegBuy},
			last:  conversionLegFixture{currency: "USD", amount: "10", leg: ConversionLegSell},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db := newAirwallexPhase2PostgresFixture(t, requiredAirwallexPhase2DatabaseURL(t))
			ctx := context.Background()
			source := loadAirwallexPhase2Source(t, db)
			repository := NewDBRepository(db)
			processor := newAirwallexPhase2Processor(t, repository, "phase2-conversion")
			groupKey := fmt.Sprintf("phase2-conversion-%d", time.Now().UnixNano())
			testCase.first.providerID = groupKey + "-" + strings.ToLower(string(testCase.first.leg))
			testCase.last.providerID = groupKey + "-" + strings.ToLower(string(testCase.last.leg))

			for index, leg := range []conversionLegFixture{testCase.first, testCase.last} {
				factID := insertAirwallexPhase2Fact(t, db, source, leg.providerID, groupKey, leg.amount, leg.currency)
				if _, err := db.ExecContext(ctx, `
UPDATE company_fund_provider_transaction_facts
SET conversion_from_currency = 'USD',
	conversion_to_currency = 'SGD',
	conversion_rate = 1.35
WHERE id = $1`, factID); err != nil {
					t.Fatalf("add conversion evidence to fact %d: %v", factID, err)
				}
				proposal := airwallexPhase2Proposal(
					source,
					"phase2:"+leg.providerID,
					leg.providerID,
					leg.amount,
					leg.currency,
				)
				proposal.MovementKind = MovementKindConversion
				proposal.Direction = DirectionInternalTransfer
				proposal.FromCompanyFundAccountID = &source.accountID
				proposal.ToCompanyFundAccountID = &source.accountID
				proposal.ConversionGroupKey = groupKey
				proposal.ConversionLeg = leg.leg
				proposal.ConversionGroupState = ConversionGroupIncomplete
				excluded := true
				proposal.AutomaticRisk.AutoExcludedFromSummary = &excluded

				enqueued, err := repository.EnqueueCompanyFundLedgerTask(ctx, CompanyFundLedgerTaskInput{
					Channel:                      ChannelAirwallex,
					ProviderAccountKey:           source.accountKey,
					Kind:                         LedgerTaskKindConversionPair,
					ProviderTransactionFactID:    factID,
					SubjectProviderTransactionID: leg.providerID,
					RelationshipReferenceType:    RelationshipReferenceSourceIDConversion,
					RelationshipReferenceKey:     groupKey,
					RelationshipGroupKey:         groupKey,
					EvidenceReference:            "postgres-integration-conversion-group",
					Proposal:                     proposal,
					RelationshipSLA:              time.Hour,
				})
				if err != nil || !enqueued.Inserted {
					t.Fatalf("enqueue conversion leg %s = %#v, %v", leg.leg, enqueued, err)
				}
				if index == 0 {
					result, err := processor.ProcessNext(ctx)
					if err != nil || result.Outcome != LedgerTaskProcessRetrying {
						t.Fatalf("orphan conversion ProcessNext() = %#v, %v", result, err)
					}
					var visible int
					if err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM company_fund_transactions
WHERE conversion_group_key = $1
  AND conversion_group_status = 'COMPLETE'`, groupKey).Scan(&visible); err != nil {
						t.Fatalf("count orphan conversion visibility: %v", err)
					}
					if visible != 0 {
						t.Fatalf("orphan conversion completed movement count = %d, want 0", visible)
					}
				}
			}

			first, err := processor.ProcessNext(ctx)
			if err != nil || first.Outcome != LedgerTaskProcessCompleted {
				t.Fatalf("paired conversion ProcessNext() = %#v, %v", first, err)
			}
			// The previously orphaned task is independently recoverable and
			// converges on the same immutable pair without another movement.
			if _, err := db.ExecContext(ctx, `
UPDATE company_fund_ledger_tasks
SET next_attempt_at = clock_timestamp()
WHERE task_kind = 'CONVERSION_PAIR' AND task_state = 'WAITING'`); err != nil {
				t.Fatalf("make orphan conversion retryable: %v", err)
			}
			second, err := processor.ProcessNext(ctx)
			if err != nil || second.Outcome != LedgerTaskProcessCompleted {
				t.Fatalf("recovered conversion ProcessNext() = %#v, %v", second, err)
			}

			rows, err := db.QueryContext(ctx, `
SELECT id, conversion_pair_transaction_id, conversion_leg
FROM company_fund_transactions
WHERE conversion_group_key = $1
  AND conversion_group_status = 'COMPLETE'
  AND transaction_direction = 'INTERNAL_TRANSFER'
  AND from_company_fund_account_id = $2
  AND to_company_fund_account_id = $2
  AND auto_excluded_from_summary
ORDER BY conversion_leg`, groupKey, source.accountID)
			if err != nil {
				t.Fatalf("query completed conversion: %v", err)
			}
			defer rows.Close()
			transactionIDs := make(map[int64]struct{}, 2)
			transactionPairs := make(map[int64]int64, 2)
			legs := make([]ConversionLeg, 0, 2)
			for rows.Next() {
				var transactionID int64
				var pairTransactionID int64
				var leg ConversionLeg
				if err := rows.Scan(&transactionID, &pairTransactionID, &leg); err != nil {
					t.Fatalf("scan completed conversion: %v", err)
				}
				transactionIDs[transactionID] = struct{}{}
				transactionPairs[transactionID] = pairTransactionID
				legs = append(legs, leg)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if len(transactionIDs) != 2 || len(legs) != 2 ||
				legs[0] != ConversionLegBuy || legs[1] != ConversionLegSell {
				t.Fatalf("completed conversion IDs=%v legs=%v", transactionIDs, legs)
			}
			for transactionID, pairTransactionID := range transactionPairs {
				if pairTransactionID == transactionID {
					t.Fatalf("conversion %d points to itself", transactionID)
				}
				if _, exists := transactionIDs[pairTransactionID]; !exists ||
					transactionPairs[pairTransactionID] != transactionID {
					t.Fatalf("conversion pair links are not reciprocal: %#v", transactionPairs)
				}
			}

			defaultDetails, err := repository.ListFinanceTransactionDetails(ctx, FinanceTransactionDetailRequest{
				Filter: FinanceTransactionFilter{
					Channels:   []TransactionSource{ChannelAirwallex},
					AccountIDs: []int64{source.accountID},
					Currencies: []string{"USD", "SGD"},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			assertAirwallexPhase2DetailsExcludeIDs(t, defaultDetails, transactionIDs)
			includedDetails, err := repository.ListFinanceTransactionDetails(ctx, FinanceTransactionDetailRequest{
				Filter: FinanceTransactionFilter{
					Channels:               []TransactionSource{ChannelAirwallex},
					AccountIDs:             []int64{source.accountID},
					Currencies:             []string{"USD", "SGD"},
					IncludeSummaryExcluded: true,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			assertAirwallexPhase2DetailsIncludeIDs(t, includedDetails, transactionIDs)
		})
	}
}

// TestAirwallexPhase2ReversalPostgresIntegration proves independent reversal
// auditability, inherited classification, net reporting, adaptive propagation,
// and the permanent priority of a manual clear on the reversal itself.
func TestAirwallexPhase2ReversalPostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL Airwallex Phase 2 coverage")
	}
	db := newAirwallexPhase2PostgresFixture(t, requiredAirwallexPhase2DatabaseURL(t))
	ctx := context.Background()
	source := loadAirwallexPhase2Source(t, db)
	repository := NewDBRepository(db)
	processor := newAirwallexPhase2Processor(t, repository, "phase2-reversal")
	level1ID, level2ID := insertAirwallexPhase2Categories(t, db, "PHASE2_REVERSAL")

	originalProviderID := fmt.Sprintf("phase2-original-%d", time.Now().UnixNano())
	originalProposal := airwallexPhase2Proposal(
		source,
		"phase2:"+originalProviderID,
		originalProviderID,
		"15",
		"USD",
	)
	originalProposal.MovementKind = MovementKindPrincipal
	originalProposal.Direction = DirectionOutflow
	originalProposal.FromCompanyFundAccountID = &source.accountID
	original, err := repository.UpsertCompanyFundTransaction(ctx, originalProposal)
	if err != nil {
		t.Fatalf("insert reversal original: %v", err)
	}
	operating := true
	include := true
	if _, err := repository.UpdateFinanceTransactionClassification(ctx, FinanceClassificationUpdate{
		TransactionID:            original.ID,
		FinanceCategoryLevel1ID:  &level1ID,
		FinanceCategoryLevel2ID:  &level2ID,
		IsOperatingIncomeExpense: &operating,
		SummaryInclusionOverride: &include,
		UpdatedBy:                "phase2-postgres-integration",
	}); err != nil {
		t.Fatalf("classify reversal original: %v", err)
	}

	wrongProviderID := fmt.Sprintf("phase2-reversal-wrong-%d", time.Now().UnixNano())
	wrongFactID := insertAirwallexPhase2Fact(t, db, source, wrongProviderID, originalProviderID, "15", "SGD")
	wrongProposal := airwallexPhase2Proposal(
		source,
		"phase2:"+wrongProviderID,
		wrongProviderID,
		"15",
		"SGD",
	)
	wrongProposal.MovementKind = MovementKindReversal
	wrongProposal.Direction = DirectionInflow
	wrongProposal.ToCompanyFundAccountID = &source.accountID
	if _, err := repository.EnqueueCompanyFundLedgerTask(ctx, CompanyFundLedgerTaskInput{
		Channel:                      ChannelAirwallex,
		ProviderAccountKey:           source.accountKey,
		Kind:                         LedgerTaskKindReversalRelationship,
		ProviderTransactionFactID:    wrongFactID,
		SubjectProviderTransactionID: wrongProviderID,
		RelationshipReferenceType:    RelationshipReferenceSourceIDReversalTarget,
		RelationshipReferenceKey:     originalProviderID,
		EvidenceReference:            "postgres-integration-wrong-reversal-target",
		Proposal:                     wrongProposal,
		RelationshipSLA:              time.Hour,
	}); err != nil {
		t.Fatalf("enqueue wrong reversal: %v", err)
	}
	wrongResult, err := processor.ProcessNext(ctx)
	if err != nil || wrongResult.Outcome != LedgerTaskProcessDeadLetter {
		t.Fatalf("wrong reversal ProcessNext() = %#v, %v", wrongResult, err)
	}
	var wrongMovements int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM company_fund_transactions WHERE provider_transaction_id = $1`,
		wrongProviderID,
	).Scan(&wrongMovements); err != nil {
		t.Fatal(err)
	}
	if wrongMovements != 0 {
		t.Fatalf("wrong reversal persisted %d movement(s)", wrongMovements)
	}

	reversalProviderID := fmt.Sprintf("phase2-reversal-%d", time.Now().UnixNano())
	factID := insertAirwallexPhase2Fact(t, db, source, reversalProviderID, originalProviderID, "15", "USD")
	reversalProposal := airwallexPhase2Proposal(
		source,
		"phase2:"+reversalProviderID,
		reversalProviderID,
		"15",
		"USD",
	)
	reversalProposal.MovementKind = MovementKindReversal
	reversalProposal.Direction = DirectionInflow
	reversalProposal.ToCompanyFundAccountID = &source.accountID
	enqueued, err := repository.EnqueueCompanyFundLedgerTask(ctx, CompanyFundLedgerTaskInput{
		Channel:                      ChannelAirwallex,
		ProviderAccountKey:           source.accountKey,
		Kind:                         LedgerTaskKindReversalRelationship,
		ProviderTransactionFactID:    factID,
		SubjectProviderTransactionID: reversalProviderID,
		RelationshipReferenceType:    RelationshipReferenceSourceIDReversalTarget,
		RelationshipReferenceKey:     originalProviderID,
		EvidenceReference:            "postgres-integration-reversal-target",
		Proposal:                     reversalProposal,
		RelationshipSLA:              time.Hour,
	})
	if err != nil || !enqueued.Inserted {
		t.Fatalf("enqueue reversal = %#v, %v", enqueued, err)
	}
	processed, err := processor.ProcessNext(ctx)
	if err != nil || processed.Outcome != LedgerTaskProcessCompleted {
		t.Fatalf("reversal ProcessNext() = %#v, %v", processed, err)
	}

	var (
		reversalID     int64
		reversalOfID   int64
		inheritedL1    int64
		inheritedL2    int64
		classification string
	)
	if err := db.QueryRowContext(ctx, `
SELECT id, reversal_of_transaction_id, finance_category_level1_id,
       finance_category_level2_id, classification_source
FROM company_fund_transactions
WHERE provider_transaction_id = $1`, reversalProviderID).Scan(
		&reversalID,
		&reversalOfID,
		&inheritedL1,
		&inheritedL2,
		&classification,
	); err != nil {
		t.Fatalf("read reversal: %v", err)
	}
	if reversalOfID != original.ID || inheritedL1 != level1ID || inheritedL2 != level2ID ||
		classification != "INHERITED_REVERSAL" {
		t.Fatalf("reversal inheritance original=%d level1=%d level2=%d source=%s",
			reversalOfID, inheritedL1, inheritedL2, classification)
	}

	summary, err := repository.GetFinanceDashboard(ctx, FinanceTransactionFilter{
		Channels:                 []TransactionSource{ChannelAirwallex},
		FinanceCategoryLevel1IDs: []int64{level1ID},
		FinanceCategoryLevel2IDs: []int64{level2ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Aggregates) != 1 || summary.Aggregates[0].TransactionCount != 2 ||
		summary.Aggregates[0].Direction != DirectionOutflow ||
		!decimal.RequireFromString(summary.Aggregates[0].Amount).IsZero() {
		t.Fatalf("reversal net summary = %#v", summary.Aggregates)
	}

	if _, err := repository.UpdateFinanceTransactionClassification(ctx, FinanceClassificationUpdate{
		TransactionID: original.ID,
		UpdatedBy:     "phase2-postgres-integration-clear-original",
	}); err != nil {
		t.Fatalf("clear original classification: %v", err)
	}
	if _, err := processor.Maintain(ctx); err != nil {
		t.Fatalf("synchronize inherited reversal: %v", err)
	}
	var clearedL1, clearedL2 sql.NullInt64
	if err := db.QueryRowContext(ctx, `
SELECT finance_category_level1_id, finance_category_level2_id
FROM company_fund_transactions
WHERE id = $1`, reversalID).Scan(&clearedL1, &clearedL2); err != nil {
		t.Fatal(err)
	}
	if clearedL1.Valid || clearedL2.Valid {
		t.Fatalf("reversal did not inherit original clear: level1=%v level2=%v", clearedL1, clearedL2)
	}

	if _, err := repository.UpdateFinanceTransactionClassification(ctx, FinanceClassificationUpdate{
		TransactionID:            reversalID,
		FinanceCategoryLevel1ID:  &level1ID,
		FinanceCategoryLevel2ID:  &level2ID,
		IsOperatingIncomeExpense: &operating,
		SummaryInclusionOverride: &include,
		UpdatedBy:                "phase2-postgres-integration-manual-reversal",
	}); err != nil {
		t.Fatalf("manually classify reversal: %v", err)
	}
	if _, err := processor.Maintain(ctx); err != nil {
		t.Fatalf("maintain manually owned reversal: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
SELECT classification_source, finance_category_level1_id, finance_category_level2_id
FROM company_fund_transactions
WHERE id = $1`, reversalID).Scan(&classification, &inheritedL1, &inheritedL2); err != nil {
		t.Fatal(err)
	}
	if classification != "MANUAL" || inheritedL1 != level1ID || inheritedL2 != level2ID {
		t.Fatalf("manual reversal was overwritten: source=%s level1=%d level2=%d",
			classification, inheritedL1, inheritedL2)
	}
}

type airwallexPhase2Source struct {
	accountID       int64
	accountKey      string
	eventID         int64
	providerEventID string
	payloadDigest   string
}

func requiredAirwallexPhase2DatabaseURL(t *testing.T) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if value == "" {
		t.Fatal("DATABASE_URL is required when Airwallex Phase 2 integration coverage is enabled")
	}
	return value
}

func loadAirwallexPhase2Source(t *testing.T, db *sql.DB) airwallexPhase2Source {
	t.Helper()
	var source airwallexPhase2Source
	if err := db.QueryRow(`
SELECT account.id, account.provider_account_key,
       event.id, event.provider_event_id, event.source_payload_digest
FROM company_fund_accounts account
JOIN company_fund_provider_events event
  ON event.channel = account.channel
 AND event.provider_account_key = account.provider_account_key
WHERE account.channel = 'AIRWALLEX'
  AND account.is_enabled
ORDER BY account.id, event.id
LIMIT 1`).Scan(
		&source.accountID,
		&source.accountKey,
		&source.eventID,
		&source.providerEventID,
		&source.payloadDigest,
	); err != nil {
		t.Fatalf("load Airwallex Phase 2 source: %v", err)
	}
	return source
}

func insertAirwallexPhase2Fact(
	t *testing.T,
	db *sql.DB,
	source airwallexPhase2Source,
	providerID string,
	sourceReference string,
	amount string,
	currency string,
) int64 {
	t.Helper()
	var factID int64
	if err := db.QueryRow(`
INSERT INTO company_fund_provider_transaction_facts (
	channel, provider_account_key, provider_transaction_id,
	provider_source_reference, fact_identity_key, fact_version,
	source_provider_event_id, source_payload_digest,
	provider_amount, provider_currency, value_scope, allocation_state,
	provider_extras
) VALUES (
	'AIRWALLEX', $1, $2, $3, $4, 1, $5, $6,
	$7::numeric, $8, 'DIRECT_ITEM', 'NOT_APPLICABLE', '{}'::jsonb
)
RETURNING id`,
		source.accountKey,
		providerID,
		sourceReference,
		"phase2-fact:"+providerID,
		source.eventID,
		source.payloadDigest,
		amount,
		currency,
	).Scan(&factID); err != nil {
		t.Fatalf("insert Airwallex Phase 2 fact %s: %v", providerID, err)
	}
	return factID
}

func airwallexPhase2Proposal(
	source airwallexPhase2Source,
	movementKey string,
	providerID string,
	amount string,
	currency string,
) TransactionUpsertInput {
	proposal := validProviderEventWorkerMovement(movementKey)
	proposal.ProviderAccountKey = source.accountKey
	proposal.ProviderTransactionID = providerID
	proposal.ProviderEventID = source.providerEventID
	proposal.ProviderMovementID = providerID
	proposal.ProviderTransactionFactID = nil
	proposal.Currency = currency
	proposal.Asset = AssetIdentity{Currency: currency}
	proposal.Amount = decimal.RequireFromString(amount)
	proposal.FirstSeenSource = TransactionSeenSourceReconciliation
	proposal.LatestProviderEventID = &source.eventID
	proposal.RawSnapshotDigest = source.payloadDigest
	proposal.Provider.Metadata.Source = ProviderSourceReconciliation
	return proposal
}

func newAirwallexPhase2Processor(
	t *testing.T,
	repository *DBRepository,
	owner string,
	feePolicyVersion ...string,
) *AirwallexLedgerTaskProcessor {
	t.Helper()
	policyVersion := "phase2-fee-v1"
	if len(feePolicyVersion) > 0 {
		policyVersion = feePolicyVersion[0]
	}
	processor, err := NewAirwallexLedgerTaskProcessor(repository, AirwallexLedgerTaskProcessorConfig{
		Owner:         owner,
		LeaseDuration: time.Minute,
		RetryDelay:    time.Millisecond,
		FeeClassification: AirwallexFeeClassificationPolicy{
			Level1Code:    "PHASE2_OPERATING_EXPENSE",
			Level2Code:    "PHASE2_FEE",
			PolicyVersion: policyVersion,
		},
		ReversalPolicyVersion: "phase2-reversal-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return processor
}

func drainAirwallexPhase2Tasks(
	t *testing.T,
	ctx context.Context,
	processor *AirwallexLedgerTaskProcessor,
) {
	t.Helper()
	for attempt := 0; attempt < 100; attempt++ {
		result, err := processor.ProcessNext(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if result.Outcome == LedgerTaskProcessIdle {
			return
		}
		if result.Outcome != LedgerTaskProcessCompleted {
			t.Fatalf("unexpected Airwallex Phase 2 drain result = %#v", result)
		}
	}
	t.Fatal("Airwallex Phase 2 task drain exceeded bounded attempts")
}

func assertAirwallexPhase2Classification(
	t *testing.T,
	db *sql.DB,
	transactionID int64,
	source string,
	policyVersion string,
) {
	t.Helper()
	var actualSource string
	var actualPolicy sql.NullString
	if err := db.QueryRow(`
SELECT classification_source, classification_policy_version
FROM company_fund_transactions
WHERE id = $1`, transactionID).Scan(&actualSource, &actualPolicy); err != nil {
		t.Fatal(err)
	}
	if actualSource != source || actualPolicy.String != policyVersion || actualPolicy.Valid != (policyVersion != "") {
		t.Fatalf("transaction %d classification source=%s policy=%v, want %s/%q",
			transactionID, actualSource, actualPolicy, source, policyVersion)
	}
}

func assertAirwallexPayoutDetails(
	t *testing.T,
	db *sql.DB,
	providerTransactionID string,
	account string,
	name string,
	wantCount int,
) {
	t.Helper()
	var actualAccount, actualName, feeCurrency, feePaidBy, swiftChargeOption sql.NullString
	var feeMatches bool
	var count int
	if err := db.QueryRow(`
SELECT MAX(to_address_or_account),
       MAX(payee_name),
       BOOL_AND(provider_reported_fee_amount = 15.58),
       MAX(provider_reported_fee_currency),
       MAX(fee_details->>'feePaidBy'),
       MAX(fee_details->>'swiftChargeOption'),
       COUNT(*)
FROM company_fund_transactions
WHERE provider_transaction_id = $1`, providerTransactionID).Scan(
		&actualAccount,
		&actualName,
		&feeMatches,
		&feeCurrency,
		&feePaidBy,
		&swiftChargeOption,
		&count,
	); err != nil {
		t.Fatal(err)
	}
	if count != wantCount || !actualAccount.Valid || actualAccount.String != account ||
		!actualName.Valid || actualName.String != name || !feeMatches ||
		!feeCurrency.Valid || feeCurrency.String != "USD" ||
		!feePaidBy.Valid || feePaidBy.String != "BENEFICIARY" ||
		!swiftChargeOption.Valid || swiftChargeOption.String != "SHARED" {
		t.Fatalf(
			"payout details account=%v name=%v fee=%t/%v payer=%v swift=%v count=%d",
			actualAccount,
			actualName,
			feeMatches,
			feeCurrency,
			feePaidBy,
			swiftChargeOption,
			count,
		)
	}
}

func insertAirwallexPhase2Categories(t *testing.T, db *sql.DB, prefix string) (int64, int64) {
	t.Helper()
	var level1ID, level2ID int64
	if err := db.QueryRow(`
INSERT INTO finance_categories (level, code, name, is_enabled)
VALUES (1, $1, $2, true)
RETURNING id`, prefix+"_L1", prefix+" level 1").Scan(&level1ID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
INSERT INTO finance_categories (level, parent_id, code, name, is_enabled)
VALUES (2, $1, $2, $3, true)
RETURNING id`, level1ID, prefix+"_L2", prefix+" level 2").Scan(&level2ID); err != nil {
		t.Fatal(err)
	}
	return level1ID, level2ID
}

func assertAirwallexPhase2DetailsExcludeIDs(
	t *testing.T,
	details []FinanceTransactionDetail,
	excluded map[int64]struct{},
) {
	t.Helper()
	for _, detail := range details {
		if _, found := excluded[detail.ID]; found {
			t.Fatalf("finance detail unexpectedly included transaction %d", detail.ID)
		}
	}
}

func assertAirwallexPhase2DetailsIncludeIDs(
	t *testing.T,
	details []FinanceTransactionDetail,
	expected map[int64]struct{},
) {
	t.Helper()
	found := make(map[int64]struct{}, len(expected))
	for _, detail := range details {
		if _, wanted := expected[detail.ID]; wanted {
			found[detail.ID] = struct{}{}
		}
	}
	if len(found) != len(expected) {
		t.Fatalf("finance detail included IDs %v, want %v", found, expected)
	}
}

func newAirwallexPhase2PostgresFixture(t *testing.T, databaseURL string) *sql.DB {
	t.Helper()
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { _ = admin.Close() })
	schema := fmt.Sprintf("airwallex_phase2_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(context.Background(), `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create Airwallex Phase 2 schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`); err != nil {
			t.Errorf("drop Airwallex Phase 2 schema: %v", err)
		}
	})
	for _, table := range []string{
		"finance_categories",
		"company_fund_accounts",
		"company_fund_account_asset_policies",
		"company_fund_provider_events",
		"company_fund_provider_transaction_facts",
		"company_fund_transactions",
		"company_fund_ledger_tasks",
		"company_fund_classification_policy_bindings",
		"company_fund_account_lifecycle_commands",
		"company_fund_account_lifecycle_audits",
		"safeheron_transaction_routing_cases",
		"safeheron_transaction_routing_case_commands",
		"safeheron_transaction_routing_case_actions",
	} {
		if _, err := admin.ExecContext(context.Background(),
			`CREATE TABLE `+schema+`.`+table+` (LIKE public.`+table+` INCLUDING ALL)`,
		); err != nil {
			t.Fatalf("clone %s: %v", table, err)
		}
	}
	for _, table := range []string{
		"company_fund_accounts",
		"company_fund_provider_events",
		"company_fund_provider_transaction_facts",
		"company_fund_transactions",
	} {
		if _, err := admin.ExecContext(context.Background(),
			`INSERT INTO `+schema+`.`+table+` SELECT * FROM public.`+table+` WHERE channel = 'AIRWALLEX'`,
		); err != nil {
			t.Fatalf("copy Airwallex %s fixtures: %v", table, err)
		}
	}
	for _, table := range []string{
		"finance_categories",
		"company_fund_accounts",
		"company_fund_account_asset_policies",
		"company_fund_provider_events",
		"company_fund_provider_transaction_facts",
		"company_fund_transactions",
		"company_fund_ledger_tasks",
		"company_fund_account_lifecycle_commands",
		"company_fund_account_lifecycle_audits",
	} {
		sequence := table + "_phase2_id_seq"
		if _, err := admin.ExecContext(context.Background(),
			`ALTER TABLE `+schema+`.`+table+` ALTER COLUMN id DROP DEFAULT;
CREATE SEQUENCE `+schema+`.`+sequence+` OWNED BY `+schema+`.`+table+`.id;
ALTER TABLE `+schema+`.`+table+` ALTER COLUMN id
  SET DEFAULT nextval('`+schema+`.`+sequence+`'::regclass);
SELECT setval(
  '`+schema+`.`+sequence+`'::regclass,
  COALESCE((SELECT MAX(id) FROM `+schema+`.`+table+`), 0) + 1,
  false
)`,
		); err != nil {
			t.Fatalf("isolate %s sequence: %v", table, err)
		}
	}
	fixture := config.Copy()
	if fixture.RuntimeParams == nil {
		fixture.RuntimeParams = make(map[string]string)
	}
	fixture.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*fixture)
	db.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = db.Close() })
	return db
}
