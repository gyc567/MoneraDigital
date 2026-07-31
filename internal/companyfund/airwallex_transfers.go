package companyfund

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrAirwallexTransferBeneficiaryUnavailable = errors.New("airwallex transfer beneficiary is unavailable")
	ErrAirwallexTransferBeneficiaryTemporary   = errors.New("airwallex transfer beneficiary is temporarily unavailable")
)

// AirwallexTransferBeneficiaryClient is the narrow provider surface required
// by PAYOUT counterparty enrichment.
type AirwallexTransferBeneficiaryClient interface {
	GetTransferBeneficiary(ctx context.Context, transferID string) (AirwallexTransferBeneficiary, error)
}

// AirwallexTransferBeneficiaryScopedClientFactory creates an immutable client
// for the exact configured Airwallex account that owns the Financial
// Transaction snapshot.
type AirwallexTransferBeneficiaryScopedClientFactory interface {
	AirwallexTransferBeneficiaryClientForScope(
		providerAccountKey string,
	) (AirwallexTransferBeneficiaryClient, error)
}

// GetTransferBeneficiary resolves the external beneficiary for one PAYOUT
// Financial Transaction source_id. It returns only the display fields owned by
// the company-fund ledger and never exposes the full provider response.
func (c *AirwallexClient) GetTransferBeneficiary(
	ctx context.Context,
	transferID string,
) (AirwallexTransferBeneficiary, error) {
	transferID, err := validateAirwallexTransferID(transferID)
	if err != nil {
		return AirwallexTransferBeneficiary{}, err
	}
	body, err := c.authenticatedGET(ctx, c.endpoint("/api/v1/transfers/"+transferID, nil))
	if err != nil {
		return AirwallexTransferBeneficiary{}, err
	}
	var response struct {
		Beneficiary struct {
			BankDetails struct {
				AccountName   string `json:"account_name"`
				AccountNumber string `json:"account_number"`
				IBAN          string `json:"iban"`
			} `json:"bank_details"`
		} `json:"beneficiary"`
	}
	if err := decodeAirwallexJSON(body, &response); err != nil {
		return AirwallexTransferBeneficiary{}, err
	}
	addressOrAccount := strings.TrimSpace(response.Beneficiary.BankDetails.AccountNumber)
	if addressOrAccount == "" {
		addressOrAccount = strings.TrimSpace(response.Beneficiary.BankDetails.IBAN)
	}
	return AirwallexTransferBeneficiary{
		AddressOrAccount: addressOrAccount,
		Name:             strings.TrimSpace(response.Beneficiary.BankDetails.AccountName),
	}, nil
}

func validateAirwallexTransferID(value string) (string, error) {
	transferID := strings.TrimSpace(value)
	if transferID == "" || len(transferID) > maxAirwallexFinancialTransactionIDBytes ||
		strings.ContainsAny(transferID, "/?#%\\") {
		return "", fmt.Errorf("airwallex transfer ID is invalid")
	}
	return transferID, nil
}

// AirwallexTransferBeneficiaryClientForScope returns a scope-isolated client
// so beneficiary reads cannot silently cross configured Airwallex accounts.
func (c *AirwallexClient) AirwallexTransferBeneficiaryClientForScope(
	providerAccountKey string,
) (AirwallexTransferBeneficiaryClient, error) {
	return c.airwallexClientForScope(providerAccountKey)
}
