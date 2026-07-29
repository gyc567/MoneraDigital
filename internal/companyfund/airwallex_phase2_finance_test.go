package companyfund

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"
)

func TestApplyAirwallexFeeClassification_UsesCodesAndPreservesManualOwnership(t *testing.T) {
	db, mock := newFinanceMockDB(t)
	defer db.Close()
	repository := NewDBRepository(db)
	policy := AirwallexFeeClassificationPolicy{
		Level1Code:    "OPERATING_EXPENSE",
		Level2Code:    "FEE",
		PolicyVersion: "airwallex-fee-v1",
	}
	mock.ExpectQuery(regexp.QuoteMeta(resolveAirwallexFeeCategoriesSQL)).
		WithArgs(policy.Level1Code, policy.Level2Code).
		WillReturnRows(sqlmock.NewRows([]string{"level1_id", "level2_id"}).AddRow(11, 22))
	mock.ExpectExec(regexp.QuoteMeta(publishAirwallexFeeClassificationBindingSQL)).
		WithArgs(int64(11), int64(22), policy.PolicyVersion).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(applyAirwallexFeeClassificationSQL)).
		WithArgs(int64(71), int64(11), int64(22), policy.PolicyVersion).
		WillReturnError(sql.ErrNoRows)

	applied, err := repository.ApplyAirwallexFeeClassification(context.Background(), 71, policy)
	if err != nil || applied {
		t.Fatalf("ApplyAirwallexFeeClassification() = %t, %v; want protected no-op", applied, err)
	}
	for _, required := range []string{
		"set_config('monera.company_fund_classification_origin', 'SYSTEM', true)",
		"classification_source IN ('UNCLASSIFIED', 'AUTO_RULE')",
		"is_operating_income_expense = TRUE",
		"summary_inclusion_override = TRUE",
	} {
		if !strings.Contains(applyAirwallexFeeClassificationSQL, required) {
			t.Fatalf("automatic fee classification SQL missing %q", required)
		}
	}
	if !strings.Contains(updateFinanceTransactionClassificationSQL, "classification_source = 'MANUAL'") ||
		!strings.Contains(updateFinanceTransactionClassificationSQL, "classification_policy_version = NULL") {
		t.Fatal("manual finance update must permanently claim classification ownership")
	}
	assertFinanceMockExpectations(t, mock)
}

func TestReversalInheritance_IsNettableAndStopsAtManualOwnership(t *testing.T) {
	for _, required := range []string{
		"reversal.classification_source <> 'MANUAL'",
		"classification_source = 'INHERITED_REVERSAL'",
		"set_config('monera.company_fund_classification_origin', 'SYSTEM', true)",
	} {
		if !strings.Contains(applyReversalClassificationInheritanceSQL, required) ||
			!strings.Contains(synchronizeReversalClassificationInheritanceSQL, required) {
			t.Fatalf("reversal inheritance SQL missing %q", required)
		}
	}
	if !strings.Contains(financeSignedAmountSQL, "-transaction.amount") ||
		!strings.Contains(financeSignedUSDValueSQL, "-transaction.usd_value") ||
		!strings.Contains(financeEffectiveDirectionSQL, "original_transaction.transaction_direction") {
		t.Fatal("finance summary must net a reversal into the original direction/category")
	}
}

func TestAirwallexLedgerTaskProcessor_KeepsLedgerAvailableWithPendingFinancePolicy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()
	processor, err := NewAirwallexLedgerTaskProcessor(NewDBRepository(db), AirwallexLedgerTaskProcessorConfig{
		Owner:         "worker-a",
		LeaseDuration: 30,
		RetryDelay:    30,
	})
	if err != nil {
		t.Fatalf("missing finance policy must not disable relationship processing: %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta(listAirwallexFeesNeedingClassificationSQL)).
		WithArgs("airwallex-fee-policy-unconfigured-v1", 100).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"provider_transaction_fact_id",
			"provider_account_key",
			"provider_transaction_id",
		}))
	maintained, err := processor.Maintain(context.Background())
	if err != nil || maintained {
		t.Fatalf("pending finance policy Maintain() = %t, %v", maintained, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAirwallexLedgerTaskProcessor_NormalizesStableFinancePolicy(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	processor, err := NewAirwallexLedgerTaskProcessor(NewDBRepository(db), AirwallexLedgerTaskProcessorConfig{
		Owner:         "phase2-policy-test",
		LeaseDuration: time.Minute,
		RetryDelay:    time.Minute,
		FeeClassification: AirwallexFeeClassificationPolicy{
			Level1Code:    " OPERATING_EXPENSE ",
			Level2Code:    " FEE ",
			PolicyVersion: " fee-v1 ",
		},
		ReversalPolicyVersion: " reversal-v1 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if processor.config.FeeClassification.Level1Code != "OPERATING_EXPENSE" ||
		processor.config.FeeClassification.Level2Code != "FEE" ||
		processor.config.FeeClassification.PolicyVersion != "fee-v1" ||
		processor.config.ReversalPolicyVersion != "reversal-v1" {
		t.Fatalf("processor policy was not canonicalized: %#v", processor.config)
	}
}

func TestValidateAirwallexConversionPairEvidenceFailsClosed(t *testing.T) {
	accountID := int64(7)
	rate := decimal.RequireFromString("1.35")
	sellAmount := decimal.RequireFromString("10")
	buyAmount := decimal.RequireFromString("13.5")
	lease := CompanyFundLedgerTaskLease{
		Channel:                      ChannelAirwallex,
		ProviderAccountKey:           "awx-account",
		ProviderTransactionFactID:    11,
		SubjectProviderTransactionID: "sell-1",
		RelationshipReferenceType:    RelationshipReferenceSourceIDConversion,
		RelationshipReferenceKey:     "conversion-1",
		RelationshipGroupKey:         "conversion-1",
		EvidenceReference:            "sandbox-conversion-v1",
		Proposal: TransactionUpsertInput{
			Direction:                DirectionInternalTransfer,
			Currency:                 "USD",
			Amount:                   sellAmount,
			ConversionLeg:            ConversionLegSell,
			FromCompanyFundAccountID: &accountID,
			ToCompanyFundAccountID:   &accountID,
		},
	}
	peer := lease
	peer.ProviderTransactionFactID = 12
	peer.SubjectProviderTransactionID = "buy-1"
	peer.Proposal.Currency = "SGD"
	peer.Proposal.Amount = buyAmount
	peer.Proposal.ConversionLeg = ConversionLegBuy
	sellFact := ProviderTransactionFact{
		ProviderTransactionID:   "sell-1",
		ProviderSourceReference: "conversion-1",
		ProviderAmount:          &sellAmount,
		ProviderCurrency:        "USD",
		ConversionFromCurrency:  "USD",
		ConversionToCurrency:    "SGD",
		ConversionRate:          &rate,
	}
	buyFact := sellFact
	buyFact.ProviderTransactionID = "buy-1"
	buyFact.ProviderAmount = &buyAmount
	buyFact.ProviderCurrency = "SGD"

	if code := validateAirwallexConversionPairEvidence(lease, peer, sellFact, buyFact); code != "" {
		t.Fatalf("valid conversion evidence rejected with %s", code)
	}

	invalidPeer := peer
	invalidPeer.Proposal.Direction = DirectionInflow
	if code := validateAirwallexConversionPairEvidence(lease, invalidPeer, sellFact, buyFact); code != "CONVERSION_ACCOUNT_DIRECTION_CONFLICT" {
		t.Fatalf("invalid direction conflict code = %q", code)
	}
	invalidBuyFact := buyFact
	invalidRate := decimal.RequireFromString("1.36")
	invalidBuyFact.ConversionRate = &invalidRate
	if code := validateAirwallexConversionPairEvidence(lease, peer, sellFact, invalidBuyFact); code != "CONVERSION_PAIR_FACT_CONFLICT" {
		t.Fatalf("invalid pair fact conflict code = %q", code)
	}
}

func TestValidateAirwallexReversalSemanticsRequiresNettableTarget(t *testing.T) {
	accountID := int64(7)
	target := airwallexRelationshipTarget{
		ID:        10,
		Key:       "principal-10",
		Kind:      MovementKindPrincipal,
		Currency:  "USD",
		Amount:    decimal.RequireFromString("25"),
		Direction: DirectionOutflow,
		FromID:    &accountID,
	}
	proposal := TransactionUpsertInput{
		Currency:               "USD",
		Amount:                 decimal.RequireFromString("25"),
		Direction:              DirectionInflow,
		ToCompanyFundAccountID: &accountID,
	}
	if code := validateAirwallexReversalSemantics(proposal, target); code != "" {
		t.Fatalf("nettable reversal rejected with %q", code)
	}
	wrongCurrency := proposal
	wrongCurrency.Currency = "SGD"
	if code := validateAirwallexReversalSemantics(wrongCurrency, target); code != "REVERSAL_VALUE_CONFLICT" {
		t.Fatalf("wrong currency conflict code = %q", code)
	}
	wrongDirection := proposal
	wrongDirection.Direction = DirectionOutflow
	if code := validateAirwallexReversalSemantics(wrongDirection, target); code != "REVERSAL_DIRECTION_ACCOUNT_CONFLICT" {
		t.Fatalf("wrong direction conflict code = %q", code)
	}
}
