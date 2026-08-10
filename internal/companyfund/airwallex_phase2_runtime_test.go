package companyfund

import (
	"context"
	"testing"
	"time"
)

func TestAirwallexPhase2Runtime_ExactFeeWaitsWithoutHalfMovement(t *testing.T) {
	config := testAirwallexRuntimeExternalConfig()
	config.Rules[0].Classification = AirwallexFinancialTransactionClassification{
		TransactionType: "FEE",
		SourceType:      "PAYMENT",
		Action:          AirwallexFinancialTransactionActionApply,
		MovementKind:    MovementKindFee,
		Direction:       DirectionOutflow,
		TransferMode:    TransferModeSingle,
		AmountField:     AirwallexFinancialAmountFieldFee,
		ExpectedSign:    AirwallexFinancialValueSignNegative,
		OccurredAtField: AirwallexFinancialOccurredAtCreated,
	}
	config.Rules[0].Counterparty = nil
	config.Rules[0].Relationship = AirwallexRuntimeRelationshipRule{
		Strategy:          AirwallexRuntimeRelationshipSourceExactParent,
		EvidenceReference: "sandbox-fee-source-id-contract",
		SLADuration:       24 * time.Hour,
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(
		config,
		&airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)},
	)
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundle() error = %v", err)
	}

	result, err := bundle.ProviderEvents.NormalizeProviderEvent(
		context.Background(),
		testAirwallexFinancialTransactionProviderEventLease("awx-usd"),
		[]byte(`{"id":"fee_1","amount":-1.23,"fee":-1.23,"net":-1.23,"created_at":"2026-07-10T01:02:03Z","currency":"USD","source_id":"payment_1","source_type":"PAYMENT","status":"SETTLED","transaction_type":"FEE"}`),
	)
	if err != nil {
		t.Fatalf("NormalizeProviderEvent() error = %v", err)
	}
	if len(result.Facts) != 1 || len(result.DeferredMovements) != 1 || len(result.Movements) != 0 {
		t.Fatalf("deferred fee result = %#v", result)
	}
	task := result.DeferredMovements[0].Task
	if task.Kind != LedgerTaskKindFeeRelationship ||
		task.RelationshipReferenceType != RelationshipReferenceSourceIDExactParent ||
		task.RelationshipReferenceKey != "payment_1" ||
		task.Proposal.ParentMovementKey != "" {
		t.Fatalf("deferred fee task = %#v", task)
	}
}

func TestAirwallexPhase2Runtime_SourceGroupOnlyFeePersistsWithoutInventingParent(t *testing.T) {
	config := testAirwallexRuntimeExternalConfig()
	config.Rules[0].Classification = AirwallexFinancialTransactionClassification{
		TransactionType: "FEE",
		SourceType:      "FEE",
		Action:          AirwallexFinancialTransactionActionApply,
		MovementKind:    MovementKindFee,
		Direction:       DirectionOutflow,
		TransferMode:    TransferModeSingle,
		AmountField:     AirwallexFinancialAmountFieldAmount,
		ExpectedSign:    AirwallexFinancialValueSignNegative,
		OccurredAtField: AirwallexFinancialOccurredAtCreated,
	}
	config.Rules[0].Counterparty = nil
	config.Rules[0].Relationship = AirwallexRuntimeRelationshipRule{
		Strategy:          AirwallexRuntimeRelationshipSourceGroupOnly,
		EvidenceReference: "sandbox-fee-source-group-contract",
		SLADuration:       24 * time.Hour,
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(
		config,
		&airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)},
	)
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundle() error = %v", err)
	}

	result, err := bundle.ProviderEvents.NormalizeProviderEvent(
		context.Background(),
		testAirwallexFinancialTransactionProviderEventLease("awx-usd"),
		[]byte(`{"id":"fee_2","amount":-0.75,"fee":0,"net":-0.75,"created_at":"2026-07-10T01:02:03Z","currency":"USD","source_id":"fee_source_1","source_type":"FEE","status":"SETTLED","transaction_type":"FEE"}`),
	)
	if err != nil {
		t.Fatalf("NormalizeProviderEvent() error = %v", err)
	}
	if len(result.Movements) != 1 || len(result.DeferredMovements) != 0 {
		t.Fatalf("group-only fee result = %#v", result)
	}
	movement := result.Movements[0]
	if movement.ParentMovementKey != "" ||
		movement.RelationshipReferenceType != RelationshipReferenceSourceIDGroupOnly ||
		movement.RelationshipGroupKey != "fee_source_1" {
		t.Fatalf("group-only fee movement = %#v", movement)
	}
}

func TestAirwallexPhase2Runtime_RejectsRelationshipStrategyWithoutEvidence(t *testing.T) {
	config := testAirwallexRuntimeExternalConfig()
	config.Rules[0].Classification.MovementKind = MovementKindFee
	config.Rules[0].Classification.AmountField = AirwallexFinancialAmountFieldFee
	config.Rules[0].Relationship = AirwallexRuntimeRelationshipRule{
		Strategy:    AirwallexRuntimeRelationshipSourceExactParent,
		SLADuration: time.Hour,
	}
	if _, err := NewAirwallexFinancialTransactionsRuntimeBundle(
		config,
		&airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)},
	); err == nil {
		t.Fatal("relationship strategy without evidence must be rejected")
	}
}

func TestAirwallexPhase2Runtime_ConversionLegsShareOnlyProvenGroupAndStayDeferred(t *testing.T) {
	accounts := []CompanyFundAccount{
		{ID: 7, Channel: AccountChannelAirwallex, ProviderAccountKey: "awx-usd", Enabled: true},
	}
	snapshot, err := buildAccountRegistrySnapshot(accounts, nil, time.Now())
	if err != nil {
		t.Fatalf("buildAccountRegistrySnapshot() error = %v", err)
	}
	rules := make([]AirwallexFinancialTransactionsRuntimeRule, 0, 2)
	for _, leg := range []struct {
		transactionType string
		currency        string
		conversionLeg   ConversionLeg
	}{
		{"CONVERSION_SELL", "USD", ConversionLegSell},
		{"CONVERSION_BUY", "SGD", ConversionLegBuy},
	} {
		rules = append(rules, AirwallexFinancialTransactionsRuntimeRule{
			EvidenceReference:  "sandbox-conversion-source-id-pair",
			ProviderAccountKey: "awx-usd",
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
				ExpectedSign:    AirwallexFinancialValueSignPositive,
				OccurredAtField: AirwallexFinancialOccurredAtCreated,
				ClientRateUse:   AirwallexFinancialClientRateUseConversionRate,
			},
			Relationship: AirwallexRuntimeRelationshipRule{
				Strategy:          AirwallexRuntimeRelationshipSourceConversion,
				EvidenceReference: "sandbox-conversion-source-id-pair",
				ConversionLeg:     leg.conversionLeg,
				FromCurrency:      "USD",
				ToCurrency:        "SGD",
				SLADuration:       24 * time.Hour,
			},
		})
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(
		AirwallexFinancialTransactionsRuntimeConfig{
			Enabled:        true,
			APIVersion:     airwallexTestAPIVersion,
			SchemaVersion:  "schema-v1",
			EventVersion:   "event-v1",
			MappingVersion: "conversion-v1",
			FactVersion:    1,
			Rules:          rules,
		},
		&airwallexProviderEventRegistryStub{snapshot: snapshot},
	)
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundle() error = %v", err)
	}
	for _, fixture := range []struct {
		transactionType string
		currency        string
		id              string
		wantLeg         ConversionLeg
	}{
		{"CONVERSION_SELL", "USD", "sell_1", ConversionLegSell},
		{"CONVERSION_BUY", "SGD", "buy_1", ConversionLegBuy},
	} {
		payload := []byte(`{"id":"` + fixture.id + `","amount":10,"fee":0,"net":10,"client_rate":1.35,"created_at":"2026-07-10T01:02:03Z","currency":"` + fixture.currency + `","currency_pair":"USDSGD","source_id":"conversion_1","source_type":"CONVERSION","status":"SETTLED","transaction_type":"` + fixture.transactionType + `"}`)
		result, err := bundle.ProviderEvents.NormalizeProviderEvent(
			context.Background(),
			testAirwallexFinancialTransactionProviderEventLease("awx-usd"),
			payload,
		)
		if err != nil {
			t.Fatalf("NormalizeProviderEvent(%s) error = %v", fixture.wantLeg, err)
		}
		if len(result.Movements) != 0 || len(result.DeferredMovements) != 1 {
			t.Fatalf("conversion %s result = %#v", fixture.wantLeg, result)
		}
		proposal := result.DeferredMovements[0].Task.Proposal
		if proposal.FromCompanyFundAccountID == nil || proposal.ToCompanyFundAccountID == nil ||
			*proposal.FromCompanyFundAccountID != accounts[0].ID ||
			*proposal.ToCompanyFundAccountID != accounts[0].ID {
			t.Fatalf("conversion %s accounts = from %#v to %#v", fixture.wantLeg, proposal.FromCompanyFundAccountID, proposal.ToCompanyFundAccountID)
		}
		if proposal.ConversionGroupKey != "conversion_1" ||
			proposal.ConversionLeg != fixture.wantLeg ||
			proposal.ConversionGroupState != ConversionGroupIncomplete ||
			proposal.AutomaticRisk.AutoExcludedFromSummary == nil ||
			!*proposal.AutomaticRisk.AutoExcludedFromSummary {
			t.Fatalf("conversion %s proposal = %#v", fixture.wantLeg, proposal)
		}
	}
}

func TestAirwallexPhase2Runtime_RejectsConversionWithoutInternalDirectionAndRateEvidence(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		direction     Direction
		clientRateUse AirwallexFinancialClientRateUse
	}{
		{
			name:          "external direction",
			direction:     DirectionOutflow,
			clientRateUse: AirwallexFinancialClientRateUseConversionRate,
		},
		{
			name:          "missing provider rate",
			direction:     DirectionInternalTransfer,
			clientRateUse: AirwallexFinancialClientRateUseNone,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			config := testAirwallexRuntimeExternalConfig()
			config.Rules[0].Classification = AirwallexFinancialTransactionClassification{
				TransactionType: "CONVERSION_SELL",
				SourceType:      "CONVERSION",
				Action:          AirwallexFinancialTransactionActionApply,
				MovementKind:    MovementKindConversion,
				Direction:       testCase.direction,
				TransferMode:    TransferModeSingle,
				AmountField:     AirwallexFinancialAmountFieldAmount,
				ExpectedSign:    AirwallexFinancialValueSignPositive,
				OccurredAtField: AirwallexFinancialOccurredAtCreated,
				ClientRateUse:   testCase.clientRateUse,
			}
			config.Rules[0].Counterparty = nil
			config.Rules[0].Relationship = AirwallexRuntimeRelationshipRule{
				Strategy:          AirwallexRuntimeRelationshipSourceConversion,
				EvidenceReference: "sandbox-conversion-source-id-pair",
				ConversionLeg:     ConversionLegSell,
				FromCurrency:      "USD",
				ToCurrency:        "SGD",
				SLADuration:       24 * time.Hour,
			}
			if _, err := NewAirwallexFinancialTransactionsRuntimeBundle(
				config,
				&airwallexProviderEventRegistryStub{
					snapshot: testAirwallexProviderEventRegistrySnapshot(t, true),
				},
			); err == nil {
				t.Fatal("invalid conversion mapping must fail closed")
			}
		})
	}
}

func TestAirwallexPhase2Runtime_ReversalWaitsForExactOriginal(t *testing.T) {
	config := testAirwallexRuntimeExternalConfig()
	config.Rules[0].Classification = AirwallexFinancialTransactionClassification{
		TransactionType: "REVERSAL",
		SourceType:      "PAYMENT",
		Action:          AirwallexFinancialTransactionActionApply,
		MovementKind:    MovementKindReversal,
		Direction:       DirectionInflow,
		TransferMode:    TransferModeSingle,
		AmountField:     AirwallexFinancialAmountFieldAmount,
		ExpectedSign:    AirwallexFinancialValueSignPositive,
		OccurredAtField: AirwallexFinancialOccurredAtCreated,
	}
	config.Rules[0].Counterparty = nil
	config.Rules[0].Relationship = AirwallexRuntimeRelationshipRule{
		Strategy:          AirwallexRuntimeRelationshipSourceReversal,
		EvidenceReference: "sandbox-reversal-source-id-contract",
		SLADuration:       24 * time.Hour,
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(
		config,
		&airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)},
	)
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundle() error = %v", err)
	}
	result, err := bundle.ProviderEvents.NormalizeProviderEvent(
		context.Background(),
		testAirwallexFinancialTransactionProviderEventLease("awx-usd"),
		[]byte(`{"id":"reversal_1","amount":9,"fee":0,"net":9,"created_at":"2026-07-10T01:02:03Z","currency":"USD","source_id":"payment_1","source_type":"PAYMENT","status":"SETTLED","transaction_type":"REVERSAL"}`),
	)
	if err != nil {
		t.Fatalf("NormalizeProviderEvent() error = %v", err)
	}
	if len(result.Movements) != 0 || len(result.DeferredMovements) != 1 {
		t.Fatalf("reversal result = %#v", result)
	}
	task := result.DeferredMovements[0].Task
	if task.Kind != LedgerTaskKindReversalRelationship ||
		task.RelationshipReferenceKey != "payment_1" ||
		task.Proposal.ReversalOfMovementKey != "" {
		t.Fatalf("reversal task = %#v", task)
	}
}
