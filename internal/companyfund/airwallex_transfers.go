package companyfund

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

const airwallexTransferAPIEnrichmentSource = "AIRWALLEX_TRANSFER_API"

var (
	ErrAirwallexTransferDetailsUnavailable = errors.New("airwallex transfer details are unavailable")
	ErrAirwallexTransferDetailsTemporary   = errors.New("airwallex transfer details are temporarily unavailable")
)

// AirwallexTransferDetailsClient is the narrow provider surface required by
// PAYOUT counterparty and fee enrichment.
type AirwallexTransferDetailsClient interface {
	GetTransferDetails(ctx context.Context, transferID string) (AirwallexTransferDetails, error)
}

// AirwallexTransferDetailsScopedClientFactory creates an immutable client for
// the exact configured Airwallex account that owns the Financial Transaction
// snapshot.
type AirwallexTransferDetailsScopedClientFactory interface {
	AirwallexTransferDetailsClientForScope(
		providerAccountKey string,
	) (AirwallexTransferDetailsClient, error)
}

// GetTransferDetails resolves the allowlisted external beneficiary and fee
// details for one PAYOUT Financial Transaction source_id.
func (c *AirwallexClient) GetTransferDetails(
	ctx context.Context,
	transferID string,
) (AirwallexTransferDetails, error) {
	transferID, err := validateAirwallexTransferID(transferID)
	if err != nil {
		return AirwallexTransferDetails{}, err
	}
	body, err := c.authenticatedGET(ctx, c.endpoint("/api/v1/transfers/"+transferID, nil))
	if err != nil {
		return AirwallexTransferDetails{}, err
	}
	var response struct {
		AmountBeneficiaryReceives json.RawMessage `json:"amount_beneficiary_receives"`
		AmountPayerPays           json.RawMessage `json:"amount_payer_pays"`
		Beneficiary               struct {
			BankDetails struct {
				AccountName   string `json:"account_name"`
				AccountNumber string `json:"account_number"`
				IBAN          string `json:"iban"`
			} `json:"bank_details"`
		} `json:"beneficiary"`
		FeeAmount         json.RawMessage `json:"fee_amount"`
		FeeCurrency       string          `json:"fee_currency"`
		FeePaidBy         string          `json:"fee_paid_by"`
		SwiftChargeOption string          `json:"swift_charge_option"`
	}
	if err := decodeAirwallexJSON(body, &response); err != nil {
		return AirwallexTransferDetails{}, err
	}
	addressOrAccount := strings.TrimSpace(response.Beneficiary.BankDetails.AccountNumber)
	if addressOrAccount == "" {
		addressOrAccount = strings.TrimSpace(response.Beneficiary.BankDetails.IBAN)
	}
	feeAmount, err := parseOptionalAirwallexJSONDecimal("transfer fee_amount", response.FeeAmount)
	if err != nil {
		return AirwallexTransferDetails{}, err
	}
	feeCurrency := strings.ToUpper(strings.TrimSpace(response.FeeCurrency))
	feePaidBy := strings.ToUpper(strings.TrimSpace(response.FeePaidBy))
	swiftChargeOption := strings.ToUpper(strings.TrimSpace(response.SwiftChargeOption))
	if err := validateAirwallexTransferFeeDetails(
		feeAmount,
		feeCurrency,
		feePaidBy,
		swiftChargeOption,
		response.AmountPayerPays,
		response.AmountBeneficiaryReceives,
	); err != nil {
		return AirwallexTransferDetails{}, err
	}
	feeDetails, err := json.Marshal(struct {
		AmountBeneficiaryReceives json.RawMessage `json:"amountBeneficiaryReceives,omitempty"`
		AmountPayerPays           json.RawMessage `json:"amountPayerPays,omitempty"`
		EnrichmentSource          string          `json:"enrichmentSource"`
		FeePaidBy                 string          `json:"feePaidBy,omitempty"`
		SwiftChargeOption         string          `json:"swiftChargeOption,omitempty"`
	}{
		AmountBeneficiaryReceives: response.AmountBeneficiaryReceives,
		AmountPayerPays:           response.AmountPayerPays,
		EnrichmentSource:          airwallexTransferAPIEnrichmentSource,
		FeePaidBy:                 feePaidBy,
		SwiftChargeOption:         swiftChargeOption,
	})
	if err != nil {
		return AirwallexTransferDetails{}, fmt.Errorf("encode Airwallex transfer fee details: %w", err)
	}
	var feeCurrencyPointer *string
	if feeCurrency != "" {
		feeCurrencyPointer = &feeCurrency
	}
	return AirwallexTransferDetails{
		Beneficiary: AirwallexTransferBeneficiary{
			AddressOrAccount: addressOrAccount,
			Name:             strings.TrimSpace(response.Beneficiary.BankDetails.AccountName),
		},
		Fee: ProviderTransactionFeeInput{
			Amount:      feeAmount,
			Currency:    feeCurrencyPointer,
			DetailsJSON: feeDetails,
		},
	}, nil
}

func validateAirwallexTransferFeeDetails(
	feeAmount *decimal.Decimal,
	feeCurrency string,
	feePaidBy string,
	swiftChargeOption string,
	amountPayerPays json.RawMessage,
	amountBeneficiaryReceives json.RawMessage,
) error {
	if (feeAmount == nil) != (feeCurrency == "") {
		return fmt.Errorf("Airwallex transfer fee amount and currency must be supplied together")
	}
	if feeAmount != nil {
		if feeAmount.IsNegative() {
			return fmt.Errorf("Airwallex transfer fee amount must be non-negative")
		}
		if !isAirwallexFiatCurrencyCode(feeCurrency) {
			return fmt.Errorf("Airwallex transfer fee currency is invalid")
		}
		if feePaidBy == "" {
			return fmt.Errorf("Airwallex transfer fee payer is required")
		}
	}
	switch feePaidBy {
	case "", "PAYER", "BENEFICIARY":
	default:
		return fmt.Errorf("unsupported Airwallex transfer fee payer %q", feePaidBy)
	}
	switch swiftChargeOption {
	case "", "PAYER", "SHARED":
	default:
		return fmt.Errorf("unsupported Airwallex SWIFT charge option %q", swiftChargeOption)
	}
	if err := validateOptionalAirwallexTransferAmount("amount_payer_pays", amountPayerPays); err != nil {
		return err
	}
	if err := validateOptionalAirwallexTransferAmount(
		"amount_beneficiary_receives",
		amountBeneficiaryReceives,
	); err != nil {
		return err
	}
	return nil
}

func isAirwallexFiatCurrencyCode(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

func validateOptionalAirwallexTransferAmount(label string, raw json.RawMessage) error {
	value, err := parseOptionalAirwallexJSONDecimal("transfer "+label, raw)
	if err != nil {
		return err
	}
	if value != nil && value.IsNegative() {
		return fmt.Errorf("Airwallex transfer %s must be non-negative", label)
	}
	return nil
}

func validateAirwallexTransferID(value string) (string, error) {
	transferID := strings.TrimSpace(value)
	if transferID == "" || len(transferID) > maxAirwallexFinancialTransactionIDBytes ||
		strings.ContainsAny(transferID, "/?#%\\") {
		return "", fmt.Errorf("airwallex transfer ID is invalid")
	}
	return transferID, nil
}

// AirwallexTransferDetailsClientForScope returns a scope-isolated client so
// Transfer detail reads cannot silently cross configured Airwallex accounts.
func (c *AirwallexClient) AirwallexTransferDetailsClientForScope(
	providerAccountKey string,
) (AirwallexTransferDetailsClient, error) {
	return c.airwallexClientForScope(providerAccountKey)
}
