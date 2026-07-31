package companyfund

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type airwallexTransferDetailsResolver struct {
	configured *AirwallexFinancialTransactionsRuntimeResolvers
	clients    AirwallexTransferDetailsScopedClientFactory
	clientMu   sync.Mutex
	scoped     map[string]AirwallexTransferDetailsClient
}

func (resolver *airwallexTransferDetailsResolver) ResolveAirwallexProviderEventCounterparty(
	ctx context.Context,
	input AirwallexProviderEventResolutionInput,
	mapping AirwallexProviderEventMapping,
) (AirwallexProviderEventCounterpartyResolution, error) {
	configured, err := resolver.configured.ResolveAirwallexProviderEventCounterparty(ctx, input, mapping)
	if err != nil {
		return AirwallexProviderEventCounterpartyResolution{}, err
	}
	rule, err := resolver.configured.lookup(input)
	if err != nil {
		return AirwallexProviderEventCounterpartyResolution{}, err
	}
	if !rule.transferDetails {
		return configured, nil
	}
	if strings.TrimSpace(input.Transaction.SourceReference) == "" {
		return AirwallexProviderEventCounterpartyResolution{}, fmt.Errorf(
			"Airwallex PAYOUT source reference is required",
		)
	}
	client, err := resolver.clientForScope(input.ProviderAccountKey)
	if err != nil {
		return AirwallexProviderEventCounterpartyResolution{}, err
	}
	details, err := client.GetTransferDetails(ctx, input.Transaction.SourceReference)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return AirwallexProviderEventCounterpartyResolution{}, err
		}
		if airwallexTransferDetailsErrorIsTemporary(err) {
			return AirwallexProviderEventCounterpartyResolution{}, fmt.Errorf(
				"%w: %v",
				ErrAirwallexTransferDetailsTemporary,
				err,
			)
		}
		return AirwallexProviderEventCounterpartyResolution{}, fmt.Errorf(
			"%w: %v",
			ErrAirwallexTransferDetailsUnavailable,
			err,
		)
	}
	if configured.Counterparty == nil && configured.CompanyProviderAccountKey == "" {
		if strings.TrimSpace(details.Beneficiary.AddressOrAccount) == "" {
			return AirwallexProviderEventCounterpartyResolution{}, ErrAirwallexTransferDetailsUnavailable
		}
		configured.Counterparty = &AirwallexCounterparty{
			AddressOrAccount: details.Beneficiary.AddressOrAccount,
			Name:             details.Beneficiary.Name,
		}
	}
	configured.Fee = cloneProviderTransactionFeeInput(details.Fee)
	return configured, nil
}

func airwallexTransferDetailsErrorIsTemporary(err error) bool {
	if errors.Is(err, ErrAirwallexNetwork) ||
		errors.Is(err, ErrAirwallexServerResponse) ||
		errors.Is(err, ErrAirwallexResponseRead) {
		return true
	}
	var responseError *AirwallexHTTPError
	if !errors.As(err, &responseError) {
		return false
	}
	switch responseError.StatusCode {
	case http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusTooEarly,
		http.StatusTooManyRequests:
		return true
	default:
		return responseError.StatusCode >= http.StatusInternalServerError
	}
}

func (resolver *airwallexTransferDetailsResolver) clientForScope(
	providerAccountKey string,
) (AirwallexTransferDetailsClient, error) {
	resolver.clientMu.Lock()
	defer resolver.clientMu.Unlock()
	if client := resolver.scoped[providerAccountKey]; client != nil {
		return client, nil
	}
	client, err := resolver.clients.AirwallexTransferDetailsClientForScope(providerAccountKey)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("Airwallex transfer details client is unavailable")
	}
	if resolver.scoped == nil {
		resolver.scoped = make(map[string]AirwallexTransferDetailsClient)
	}
	resolver.scoped[providerAccountKey] = client
	return client, nil
}
