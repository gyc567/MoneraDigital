package companyfund

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAirwallexFinancialTransactionsRuntimeBundle_DisabledConfigurationIsExplicit(t *testing.T) {
	config, err := ParseAirwallexFinancialTransactionsRuntimeConfigJSON(nil)
	if err != nil {
		t.Fatalf("ParseAirwallexFinancialTransactionsRuntimeConfigJSON() error = %v", err)
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(config, nil)
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundle() error = %v", err)
	}
	if bundle.Enabled || bundle.ProviderEvents != nil || bundle.FinancialTransactions != nil || bundle.Resolvers != nil {
		t.Fatalf("disabled bundle = %#v, want no active runtime surface", bundle)
	}

	invalidDisabled := AirwallexFinancialTransactionsRuntimeConfig{
		Enabled:    false,
		APIVersion: airwallexTestAPIVersion,
	}
	if _, err := NewAirwallexFinancialTransactionsRuntimeBundle(invalidDisabled, nil); err == nil {
		t.Fatal("disabled configuration carrying a version must be rejected")
	}
}

func TestParseAirwallexFinancialTransactionsRuntimeConfigJSON_IsStrict(t *testing.T) {
	valid := `{
  "enabled": true,
  "api_version": "2025-04-29",
  "schema_version": "schema-v1",
  "event_version": "event-v1",
  "mapping_version": "mapping-v1",
  "fact_version": 1,
  "rules": [{
    "evidence_reference": "sandbox-fixture-1",
    "provider_account_key": "awx-usd",
    "currency": "usd",
    "status": "settled",
    "classification": {
      "transaction_type": "fixture_credit",
      "source_type": "fixture_source",
      "action": "APPLY",
      "movement_kind": "PRINCIPAL",
      "direction": "INFLOW",
      "transfer_mode": "SINGLE",
      "amount_field": "AMOUNT",
      "expected_sign": "POSITIVE",
      "occurred_at_field": "CREATED_AT"
    }
  }]
}`
	config, err := ParseAirwallexFinancialTransactionsRuntimeConfigJSON([]byte(valid))
	if err != nil {
		t.Fatalf("ParseAirwallexFinancialTransactionsRuntimeConfigJSON() error = %v", err)
	}
	if !config.Enabled || len(config.Rules) != 1 || config.Rules[0].Currency != "usd" || config.Rules[0].Status != "settled" {
		t.Fatalf("decoded configuration = %#v", config)
	}
	if _, err := NewAirwallexFinancialTransactionsRuntimeBundle(config, &airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)}); err != nil {
		t.Fatalf("validated parsed configuration rejected: %v", err)
	}

	for _, raw := range []string{
		`{"api_version":"2025-04-29"}`,
		`{"enabled":true,"unknown":true}`,
		strings.Replace(valid, `"fixture_source"`, `"fixture_source","source_id":"must-not-be-accepted"`, 1),
		valid + ` {}`,
	} {
		if _, err := ParseAirwallexFinancialTransactionsRuntimeConfigJSON([]byte(raw)); err == nil {
			t.Fatalf("ParseAirwallexFinancialTransactionsRuntimeConfigJSON(%s) error = nil, want strict rejection", raw)
		}
	}
}

func TestAirwallexFinancialTransactionsRuntimeBundle_MapsOnlyApprovedExternalSnapshot(t *testing.T) {
	registry := &airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(testAirwallexRuntimeExternalConfig(), registry)
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundle() error = %v", err)
	}
	if !bundle.Enabled || bundle.ProviderEvents == nil || bundle.FinancialTransactions == nil || bundle.Resolvers == nil || bundle.Resolvers.RuleCount() != 1 {
		t.Fatalf("runtime bundle = %#v", bundle)
	}

	lease := testAirwallexFinancialTransactionProviderEventLease("awx-usd")
	result, err := bundle.ProviderEvents.NormalizeProviderEvent(context.Background(), lease, testAirwallexFinancialTransactionSnapshotPayload())
	if err != nil {
		t.Fatalf("NormalizeProviderEvent() error = %v", err)
	}
	if err := result.validate(); err != nil || len(result.Movements) != 1 || len(result.Facts) != 1 {
		t.Fatalf("NormalizeProviderEvent() = %#v, validate=%v", result, err)
	}
	movement := result.Movements[0]
	if movement.Direction != DirectionInflow || movement.ToCompanyFundAccountID == nil || *movement.ToCompanyFundAccountID != 7 ||
		movement.FromCompanyFundAccountID != nil || movement.ParentMovementKey != "" || movement.ReversalOfMovementKey != "" || movement.ConversionGroupKey != "" {
		t.Fatalf("approved external movement = %#v", movement)
	}
	if movement.ProviderDisplay.PayerName == nil || *movement.ProviderDisplay.PayerName != "Approved manual payer" {
		t.Fatalf("manual counterparty was not copied into display input: %#v", movement.ProviderDisplay)
	}

	account, found := registry.snapshot.LookupAirwallex("awx-usd")
	if !found {
		t.Fatal("test configured account was not found")
	}
	resolution := AirwallexProviderEventResolutionInput{
		ProviderAccountKey: "awx-usd",
		ConfiguredAccount:  account,
		Transaction: AirwallexProviderEventTransactionContext{
			FinancialTransactionID: "ft_123",
			TransactionType:        "DEPOSIT_CREDIT",
			SourceType:             "BANK_FEED",
			Currency:               "USD",
			Status:                 "SETTLED",
		},
	}
	first, err := bundle.Resolvers.ResolveAirwallexProviderEventCounterparty(context.Background(), resolution, AirwallexProviderEventMapping{})
	if err != nil || first.Counterparty == nil {
		t.Fatalf("ResolveAirwallexProviderEventCounterparty() = %#v, %v", first, err)
	}
	first.Counterparty.Name = "mutated caller copy"
	second, err := bundle.Resolvers.ResolveAirwallexProviderEventCounterparty(context.Background(), resolution, AirwallexProviderEventMapping{})
	if err != nil || second.Counterparty == nil || second.Counterparty.Name != "Approved manual payer" {
		t.Fatalf("runtime resolver leaked mutable counterparty state: %#v, %v", second, err)
	}
}

func TestAirwallexFinancialTransactionsRuntimeBundle_ResolvesPayoutBeneficiaryFromTransferAPI(t *testing.T) {
	registry := &airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)}
	config := testAirwallexRuntimePayoutConfig()
	factory := &airwallexTransferBeneficiaryFactoryStub{
		client: &airwallexTransferBeneficiaryClientStub{
			beneficiary: AirwallexTransferBeneficiary{
				AddressOrAccount: "1234567890",
				Name:             "Ada Recipient",
			},
		},
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundleWithTransferBeneficiaryClient(
		config,
		registry,
		factory,
	)
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundleWithTransferBeneficiaryClient() error = %v", err)
	}

	payload := []byte(`{"id":"ft_payout_1","amount":-19.75,"fee":0,"net":-19.75,"created_at":"2026-07-10T03:00:00Z","currency":"USD","source_id":"transfer_1","source_type":"PAYOUT","status":"SETTLED","transaction_type":"PAYOUT"}`)
	result, err := bundle.ProviderEvents.NormalizeProviderEvent(
		context.Background(),
		testAirwallexFinancialTransactionProviderEventLease("awx-usd"),
		payload,
	)
	if err != nil {
		t.Fatalf("NormalizeProviderEvent() error = %v", err)
	}
	if len(result.Movements) != 1 {
		t.Fatalf("movement count = %d, want 1", len(result.Movements))
	}
	movement := result.Movements[0]
	if movement.ProviderDisplay.To.AddressOrAccount == nil ||
		*movement.ProviderDisplay.To.AddressOrAccount != "1234567890" ||
		movement.ProviderDisplay.PayeeName == nil ||
		*movement.ProviderDisplay.PayeeName != "Ada Recipient" {
		t.Fatalf("payout counterparty display = %#v", movement.ProviderDisplay)
	}
	if factory.scope != "awx-usd" || factory.client.transferID != "transfer_1" {
		t.Fatalf("resolver scope=%q transfer=%q, want awx-usd/transfer_1", factory.scope, factory.client.transferID)
	}
	secondPayload := []byte(`{"id":"ft_payout_2","amount":-5,"fee":0,"net":-5,"created_at":"2026-07-10T03:01:00Z","currency":"USD","source_id":"transfer_2","source_type":"PAYOUT","status":"SETTLED","transaction_type":"PAYOUT"}`)
	if _, err := bundle.ProviderEvents.NormalizeProviderEvent(
		context.Background(),
		testAirwallexFinancialTransactionProviderEventLease("awx-usd"),
		secondPayload,
	); err != nil {
		t.Fatalf("second NormalizeProviderEvent() error = %v", err)
	}
	if factory.calls != 1 {
		t.Fatalf("scoped client factory calls = %d, want one cached client", factory.calls)
	}
}

func TestAirwallexFinancialTransactionsRuntimeBundle_DoesNotFetchBeneficiaryForTerminalPayoutRule(t *testing.T) {
	config := testAirwallexRuntimePayoutConfig()
	config.Rules[0].Classification.Action = AirwallexFinancialTransactionActionQuarantine
	config.Rules[0].Classification.Reason = "terminal payout fixture"
	config.Rules[0].Classification.MovementKind = ""
	config.Rules[0].Classification.Direction = ""
	config.Rules[0].Classification.TransferMode = ""
	config.Rules[0].Classification.AmountField = ""
	config.Rules[0].Classification.ExpectedSign = ""
	config.Rules[0].Classification.OccurredAtField = ""
	factory := &airwallexTransferBeneficiaryFactoryStub{
		client: &airwallexTransferBeneficiaryClientStub{},
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundleWithTransferBeneficiaryClient(
		config,
		&airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)},
		factory,
	)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"id":"ft_payout_terminal","amount":-1,"fee":0,"net":-1,"created_at":"2026-07-10T03:00:00Z","currency":"USD","source_id":"transfer_terminal","source_type":"PAYOUT","status":"SETTLED","transaction_type":"PAYOUT"}`)
	if _, err := bundle.ProviderEvents.NormalizeProviderEvent(
		context.Background(),
		testAirwallexFinancialTransactionProviderEventLease("awx-usd"),
		payload,
	); !errors.Is(err, ErrProviderEventPermanent) {
		t.Fatalf("terminal payout error = %v, want permanent quarantine", err)
	}
	if factory.calls != 0 {
		t.Fatalf("terminal payout beneficiary factory calls = %d, want 0", factory.calls)
	}
}

func TestAirwallexFinancialTransactionsRuntimeBundle_RetriesOnlyTemporaryBeneficiaryFailures(t *testing.T) {
	client := &airwallexTransferBeneficiaryClientStub{err: ErrAirwallexNetwork}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundleWithTransferBeneficiaryClient(
		testAirwallexRuntimePayoutConfig(),
		&airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)},
		&airwallexTransferBeneficiaryFactoryStub{client: client},
	)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"id":"ft_payout_error","amount":-1,"fee":0,"net":-1,"created_at":"2026-07-10T03:00:00Z","currency":"USD","source_id":"transfer_error","source_type":"PAYOUT","status":"SETTLED","transaction_type":"PAYOUT"}`)
	if _, err := bundle.ProviderEvents.NormalizeProviderEvent(
		context.Background(),
		testAirwallexFinancialTransactionProviderEventLease("awx-usd"),
		payload,
	); !errors.Is(err, ErrAirwallexTransferBeneficiaryTemporary) ||
		errors.Is(err, ErrProviderEventPermanent) {
		t.Fatalf("network beneficiary error = %v, want temporary provider failure", err)
	}

	client.err = &AirwallexHTTPError{StatusCode: 404}
	if _, err := bundle.ProviderEvents.NormalizeProviderEvent(
		context.Background(),
		testAirwallexFinancialTransactionProviderEventLease("awx-usd"),
		payload,
	); !errors.Is(err, ErrProviderEventPermanent) {
		t.Fatalf("404 beneficiary error = %v, want permanent failure", err)
	}
}

func TestAirwallexTransferBeneficiaryErrorIsTemporary(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "network", err: ErrAirwallexNetwork, want: true},
		{name: "server classification", err: ErrAirwallexServerResponse, want: true},
		{name: "response read", err: ErrAirwallexResponseRead, want: true},
		{name: "request timeout", err: &AirwallexHTTPError{StatusCode: http.StatusRequestTimeout}, want: true},
		{name: "conflict", err: &AirwallexHTTPError{StatusCode: http.StatusConflict}, want: true},
		{name: "too early", err: &AirwallexHTTPError{StatusCode: http.StatusTooEarly}, want: true},
		{name: "rate limited", err: &AirwallexHTTPError{StatusCode: http.StatusTooManyRequests}, want: true},
		{name: "server status", err: &AirwallexHTTPError{StatusCode: http.StatusBadGateway}, want: true},
		{name: "unauthorized", err: &AirwallexHTTPError{StatusCode: http.StatusUnauthorized}},
		{name: "forbidden", err: &AirwallexHTTPError{StatusCode: http.StatusForbidden}},
		{name: "not found", err: &AirwallexHTTPError{StatusCode: http.StatusNotFound}},
		{name: "malformed response", err: ErrAirwallexMalformedResponse},
		{name: "response too large", err: ErrAirwallexResponseTooLarge},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := airwallexTransferBeneficiaryErrorIsTemporary(testCase.err); got != testCase.want {
				t.Fatalf("airwallexTransferBeneficiaryErrorIsTemporary(%v) = %t, want %t",
					testCase.err, got, testCase.want)
			}
		})
	}
}

func TestAirwallexFinancialTransactionsRuntimeBundle_QuarantinesUnknownContext(t *testing.T) {
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(
		testAirwallexRuntimeExternalConfig(),
		&airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)},
	)
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundle() error = %v", err)
	}
	lease := testAirwallexFinancialTransactionProviderEventLease("awx-usd")
	for _, payload := range [][]byte{
		[]byte(strings.Replace(string(testAirwallexFinancialTransactionSnapshotPayload()), `"SETTLED"`, `"PENDING"`, 1)),
		[]byte(strings.Replace(string(testAirwallexFinancialTransactionSnapshotPayload()), `"DEPOSIT_CREDIT"`, `"UNAPPROVED_TYPE"`, 1)),
	} {
		result, err := bundle.ProviderEvents.NormalizeProviderEvent(context.Background(), lease, payload)
		if !errors.Is(err, ErrProviderEventPermanent) || result.Ignored || len(result.Movements) != 0 || len(result.Facts) != 0 {
			t.Fatalf("unapproved Financial Transactions context = %#v, %v; want permanent quarantine", result, err)
		}
	}
}

func TestAirwallexFinancialTransactionsRuntimeBundle_UsesExactTerminalIgnoreRuleBeforeResolverLookup(t *testing.T) {
	config := testAirwallexRuntimeExternalConfig()
	config.Rules[0].Classification = AirwallexFinancialTransactionClassification{
		TransactionType: "FIXTURE_RESERVE_HOLD",
		SourceType:      "FIXTURE_RESERVE_SOURCE",
		Action:          AirwallexFinancialTransactionActionIgnore,
		Reason:          "SANDBOX_APPROVED_NON_CASH_HOLD",
	}
	config.Rules[0].Counterparty = nil
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(
		config,
		&airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)},
	)
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundle() error = %v", err)
	}
	lease := testAirwallexFinancialTransactionProviderEventLease("awx-usd")
	payload := []byte(strings.NewReplacer(
		`"DEPOSIT_CREDIT"`, `"FIXTURE_RESERVE_HOLD"`,
		`"BANK_FEED"`, `"FIXTURE_RESERVE_SOURCE"`,
	).Replace(string(testAirwallexFinancialTransactionSnapshotPayload())))
	result, err := bundle.ProviderEvents.NormalizeProviderEvent(context.Background(), lease, payload)
	if err != nil || !result.Ignored || len(result.Movements) != 0 || len(result.Facts) != 0 {
		t.Fatalf("exact terminal IGNORE rule = %#v, %v; want ignored result", result, err)
	}

	// The terminal classification is still exact. Changing only status must
	// fail at the resolver boundary instead of broadly ignoring the line.
	pendingPayload := []byte(strings.Replace(string(payload), `"SETTLED"`, `"PENDING"`, 1))
	result, err = bundle.ProviderEvents.NormalizeProviderEvent(context.Background(), lease, pendingPayload)
	if !errors.Is(err, ErrProviderEventPermanent) || result.Ignored {
		t.Fatalf("unapproved terminal context = %#v, %v; want permanent quarantine", result, err)
	}
}

func TestAirwallexFinancialTransactionsRuntimeBundle_MapsExplicitInternalConfiguredAccounts(t *testing.T) {
	accounts := []CompanyFundAccount{
		{ID: 7, Channel: AccountChannelAirwallex, ProviderAccountKey: "awx-usd", AccountName: "USD", Enabled: true},
		{ID: 8, Channel: AccountChannelAirwallex, ProviderAccountKey: "awx-sgd", AccountName: "SGD", Enabled: true},
	}
	snapshot, err := buildAccountRegistrySnapshot(accounts, nil, time.Date(2026, time.July, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildAccountRegistrySnapshot() error = %v", err)
	}
	config := AirwallexFinancialTransactionsRuntimeConfig{
		Enabled:        true,
		APIVersion:     airwallexTestAPIVersion,
		SchemaVersion:  "schema-v1",
		EventVersion:   "event-v1",
		MappingVersion: "mapping-v1",
		FactVersion:    1,
		Rules: []AirwallexFinancialTransactionsRuntimeRule{{
			EvidenceReference:                     "sandbox-internal-transfer-fixture",
			ProviderAccountKey:                    "awx-usd",
			Currency:                              "USD",
			Status:                                "SETTLED",
			ConfiguredAccountSide:                 AirwallexConfiguredAccountSideFrom,
			CounterpartyCompanyProviderAccountKey: "awx-sgd",
			Classification: AirwallexFinancialTransactionClassification{
				TransactionType: "FIXTURE_INTERNAL_DEBIT",
				SourceType:      "FIXTURE_INTERNAL_SOURCE",
				Action:          AirwallexFinancialTransactionActionApply,
				MovementKind:    MovementKindPrincipal,
				Direction:       DirectionInternalTransfer,
				TransferMode:    TransferModeSingle,
				AmountField:     AirwallexFinancialAmountFieldAmount,
				ExpectedSign:    AirwallexFinancialValueSignPositive,
				OccurredAtField: AirwallexFinancialOccurredAtCreated,
			},
		}},
	}
	bundle, err := NewAirwallexFinancialTransactionsRuntimeBundle(config, &airwallexProviderEventRegistryStub{snapshot: snapshot})
	if err != nil {
		t.Fatalf("NewAirwallexFinancialTransactionsRuntimeBundle() error = %v", err)
	}
	lease := testAirwallexFinancialTransactionProviderEventLease("awx-usd")
	payload := []byte(`{"id":"ft_internal_1","amount":12.34,"fee":0,"net":12.34,"created_at":"2026-07-10T01:02:03Z","currency":"USD","source_id":"untrusted-source-id","source_type":"FIXTURE_INTERNAL_SOURCE","status":"SETTLED","transaction_type":"FIXTURE_INTERNAL_DEBIT"}`)
	result, err := bundle.ProviderEvents.NormalizeProviderEvent(context.Background(), lease, payload)
	if err != nil {
		t.Fatalf("NormalizeProviderEvent() error = %v", err)
	}
	if len(result.Movements) != 1 {
		t.Fatalf("NormalizeProviderEvent() = %#v, want one internal movement", result)
	}
	movement := result.Movements[0]
	if movement.Direction != DirectionInternalTransfer || movement.FromCompanyFundAccountID == nil || movement.ToCompanyFundAccountID == nil ||
		*movement.FromCompanyFundAccountID != 7 || *movement.ToCompanyFundAccountID != 8 || movement.ParentMovementKey != "" || movement.ReversalOfMovementKey != "" {
		t.Fatalf("explicit internal configured account movement = %#v", movement)
	}
}

func TestAirwallexFinancialTransactionsRuntimeBundle_RejectsUnsafeGenericRelationshipAndMappingRules(t *testing.T) {
	base := testAirwallexRuntimeExternalConfig()
	tests := []struct {
		name   string
		mutate func(*AirwallexFinancialTransactionsRuntimeConfig)
	}{
		{
			name: "fee requires a dedicated relation resolver",
			mutate: func(config *AirwallexFinancialTransactionsRuntimeConfig) {
				config.Rules[0].Classification.MovementKind = MovementKindFee
				config.Rules[0].Classification.AmountField = AirwallexFinancialAmountFieldFee
			},
		},
		{
			name: "wildcard source type",
			mutate: func(config *AirwallexFinancialTransactionsRuntimeConfig) {
				config.Rules[0].Classification.SourceType = "*"
			},
		},
		{
			name: "missing review evidence",
			mutate: func(config *AirwallexFinancialTransactionsRuntimeConfig) {
				config.Rules[0].EvidenceReference = ""
			},
		},
		{
			name: "external mapping smuggles internal key",
			mutate: func(config *AirwallexFinancialTransactionsRuntimeConfig) {
				config.Rules[0].CounterpartyCompanyProviderAccountKey = "awx-other"
			},
		},
		{
			name: "conflicting classification for the same type source",
			mutate: func(config *AirwallexFinancialTransactionsRuntimeConfig) {
				conflicting := config.Rules[0]
				conflicting.ProviderAccountKey = "awx-other"
				conflicting.Classification.Direction = DirectionOutflow
				config.Rules = append(config.Rules, conflicting)
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			config := cloneAirwallexRuntimeConfig(base)
			testCase.mutate(&config)
			if _, err := NewAirwallexFinancialTransactionsRuntimeBundle(config, &airwallexProviderEventRegistryStub{snapshot: testAirwallexProviderEventRegistrySnapshot(t, true)}); err == nil {
				t.Fatal("NewAirwallexFinancialTransactionsRuntimeBundle() error = nil, want unsafe mapping rejected")
			}
		})
	}
}

func testAirwallexRuntimeExternalConfig() AirwallexFinancialTransactionsRuntimeConfig {
	return AirwallexFinancialTransactionsRuntimeConfig{
		Enabled:        true,
		APIVersion:     airwallexTestAPIVersion,
		SchemaVersion:  "schema-v1",
		EventVersion:   "event-v1",
		MappingVersion: "mapping-v1",
		FactVersion:    1,
		Rules: []AirwallexFinancialTransactionsRuntimeRule{{
			EvidenceReference:  "sandbox-approved-deposit-credit-fixture",
			ProviderAccountKey: "awx-usd",
			Currency:           "USD",
			Status:             "SETTLED",
			Classification: AirwallexFinancialTransactionClassification{
				TransactionType: "DEPOSIT_CREDIT",
				SourceType:      "BANK_FEED",
				Action:          AirwallexFinancialTransactionActionApply,
				MovementKind:    MovementKindPrincipal,
				Direction:       DirectionInflow,
				TransferMode:    TransferModeSingle,
				AmountField:     AirwallexFinancialAmountFieldAmount,
				ExpectedSign:    AirwallexFinancialValueSignPositive,
				OccurredAtField: AirwallexFinancialOccurredAtCreated,
			},
			Counterparty: &AirwallexRuntimeManualCounterparty{
				EvidenceReference: "finance-approved-payer-display",
				Name:              "Approved manual payer",
			},
		}},
	}
}

func testAirwallexRuntimePayoutConfig() AirwallexFinancialTransactionsRuntimeConfig {
	return AirwallexFinancialTransactionsRuntimeConfig{
		Enabled:        true,
		APIVersion:     airwallexTestAPIVersion,
		SchemaVersion:  "schema-v1",
		EventVersion:   "event-v1",
		MappingVersion: "mapping-v1",
		FactVersion:    1,
		Rules: []AirwallexFinancialTransactionsRuntimeRule{{
			EvidenceReference:  "sandbox-approved-payout-fixture",
			ProviderAccountKey: "awx-usd",
			Currency:           "USD",
			Status:             "SETTLED",
			Classification: AirwallexFinancialTransactionClassification{
				TransactionType: "PAYOUT",
				SourceType:      "PAYOUT",
				Action:          AirwallexFinancialTransactionActionApply,
				MovementKind:    MovementKindPrincipal,
				Direction:       DirectionOutflow,
				TransferMode:    TransferModeSingle,
				AmountField:     AirwallexFinancialAmountFieldAmount,
				ExpectedSign:    AirwallexFinancialValueSignNegative,
				OccurredAtField: AirwallexFinancialOccurredAtCreated,
			},
		}},
	}
}

type airwallexTransferBeneficiaryFactoryStub struct {
	scope  string
	client *airwallexTransferBeneficiaryClientStub
	err    error
	calls  int
}

func (stub *airwallexTransferBeneficiaryFactoryStub) AirwallexTransferBeneficiaryClientForScope(
	providerAccountKey string,
) (AirwallexTransferBeneficiaryClient, error) {
	stub.scope = providerAccountKey
	stub.calls++
	return stub.client, stub.err
}

type airwallexTransferBeneficiaryClientStub struct {
	transferID  string
	beneficiary AirwallexTransferBeneficiary
	err         error
}

func (stub *airwallexTransferBeneficiaryClientStub) GetTransferBeneficiary(
	_ context.Context,
	transferID string,
) (AirwallexTransferBeneficiary, error) {
	stub.transferID = transferID
	return stub.beneficiary, stub.err
}

func cloneAirwallexRuntimeConfig(source AirwallexFinancialTransactionsRuntimeConfig) AirwallexFinancialTransactionsRuntimeConfig {
	copy := source
	copy.Rules = make([]AirwallexFinancialTransactionsRuntimeRule, len(source.Rules))
	for index, rule := range source.Rules {
		copy.Rules[index] = rule
		if rule.Counterparty != nil {
			counterparty := *rule.Counterparty
			copy.Rules[index].Counterparty = &counterparty
		}
	}
	return copy
}
