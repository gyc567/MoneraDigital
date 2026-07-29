package companyfund

import (
	"context"
	"fmt"
	"strings"
)

// ValidateAirwallexAccountIdentity uses a fresh account-scoped token so
// candidate validation cannot mutate or reuse the current ingestion scope.
func (c *AirwallexClient) ValidateAirwallexAccountIdentity(
	ctx context.Context,
	providerAccountKey string,
) (AirwallexProviderIdentitySummary, error) {
	if c == nil || c.baseURL == nil || c.httpClient == nil || c.now == nil {
		return AirwallexProviderIdentitySummary{}, fmt.Errorf("airwallex client is not configured")
	}
	if providerAccountKey == "" || providerAccountKey != strings.TrimSpace(providerAccountKey) {
		return AirwallexProviderIdentitySummary{}, fmt.Errorf("Airwallex provider account key must be exact and have no surrounding whitespace")
	}

	scopedClient, err := c.AirwallexFinancialTransactionsClientForScope(providerAccountKey)
	if err != nil {
		return AirwallexProviderIdentitySummary{}, err
	}
	scoped, ok := scopedClient.(*AirwallexClient)
	if !ok {
		return AirwallexProviderIdentitySummary{}, fmt.Errorf("Airwallex scoped identity client is unavailable")
	}
	body, err := scoped.authenticatedGET(ctx, scoped.endpoint("/api/v1/account", nil))
	if err != nil {
		return AirwallexProviderIdentitySummary{}, err
	}
	var response struct {
		ID          string `json:"id"`
		AccountName string `json:"account_name"`
	}
	if err := decodeAirwallexJSON(body, &response); err != nil {
		return AirwallexProviderIdentitySummary{}, err
	}
	if response.ID != providerAccountKey {
		return AirwallexProviderIdentitySummary{}, fmt.Errorf("Airwallex provider account identity mismatch")
	}
	return AirwallexProviderIdentitySummary{
		ProviderAccountID: response.ID,
		AccountName:       strings.TrimSpace(response.AccountName),
	}, nil
}

var _ AirwallexAccountIdentityValidator = (*AirwallexClient)(nil)
