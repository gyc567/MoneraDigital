package companyfund

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type airwallexTransferCounterpartyResolver struct {
	configured *AirwallexFinancialTransactionsRuntimeResolvers
	clients    AirwallexTransferBeneficiaryScopedClientFactory
	clientMu   sync.Mutex
	scoped     map[string]AirwallexTransferBeneficiaryClient
}

func (resolver *airwallexTransferCounterpartyResolver) ResolveAirwallexProviderEventCounterparty(
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
	if configured.Counterparty != nil || configured.CompanyProviderAccountKey != "" ||
		!rule.transferBeneficiary {
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
	beneficiary, err := client.GetTransferBeneficiary(ctx, input.Transaction.SourceReference)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return AirwallexProviderEventCounterpartyResolution{}, err
		}
		if airwallexTransferBeneficiaryErrorIsTemporary(err) {
			return AirwallexProviderEventCounterpartyResolution{}, fmt.Errorf(
				"%w: %v",
				ErrAirwallexTransferBeneficiaryTemporary,
				err,
			)
		}
		return AirwallexProviderEventCounterpartyResolution{}, fmt.Errorf(
			"%w: %v",
			ErrAirwallexTransferBeneficiaryUnavailable,
			err,
		)
	}
	if strings.TrimSpace(beneficiary.AddressOrAccount) == "" {
		return AirwallexProviderEventCounterpartyResolution{}, ErrAirwallexTransferBeneficiaryUnavailable
	}
	return AirwallexProviderEventCounterpartyResolution{
		Counterparty: &AirwallexCounterparty{
			AddressOrAccount: beneficiary.AddressOrAccount,
			Name:             beneficiary.Name,
		},
	}, nil
}

func airwallexTransferBeneficiaryErrorIsTemporary(err error) bool {
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

func (resolver *airwallexTransferCounterpartyResolver) clientForScope(
	providerAccountKey string,
) (AirwallexTransferBeneficiaryClient, error) {
	resolver.clientMu.Lock()
	defer resolver.clientMu.Unlock()
	if client := resolver.scoped[providerAccountKey]; client != nil {
		return client, nil
	}
	client, err := resolver.clients.AirwallexTransferBeneficiaryClientForScope(providerAccountKey)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("Airwallex transfer beneficiary client is unavailable")
	}
	if resolver.scoped == nil {
		resolver.scoped = make(map[string]AirwallexTransferBeneficiaryClient)
	}
	resolver.scoped[providerAccountKey] = client
	return client, nil
}
