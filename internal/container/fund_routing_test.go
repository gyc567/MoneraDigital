package container

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"monera-digital/internal/fundrouting"
	"monera-digital/internal/safeheron"
)

type routingStatusSafeheronClient struct {
	safeheron.SafeheronClient
}

func (routingStatusSafeheronClient) ListTransactions(
	context.Context,
	safeheron.TransactionHistoryRequest,
) ([]safeheron.TransactionSnapshot, error) {
	return nil, nil
}

func (routingStatusSafeheronClient) LookupTransaction(
	context.Context,
	safeheron.TransactionLookup,
) (*safeheron.TransactionSnapshot, error) {
	return nil, nil
}

func TestFinalizeSafeheronRoutingRespectsGlobalWorkerSwitch(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c := &Container{
		DB:                   db,
		SafeheronRoutingMode: fundrouting.ModeRoutingAuthoritative,
		companyFundRuntimeConfig: companyFundRuntimeConfig{
			StartBackgroundWorkers: false,
		},
	}
	finalizeSafeheronRouting(c)
	if c.FundRoutingWorker != nil || c.FundRoutingProjectionWorker != nil || c.FundRoutingReconciler != nil {
		t.Fatal("routing workers must remain disabled while the global worker switch is off")
	}
}

func TestFinalizeSafeheronRoutingWiresProviderStatusFallbackWhenLookupIsAvailable(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &Container{
		DB:                      db,
		SafeheronClient:         routingStatusSafeheronClient{},
		SafeheronRoutingMode:    fundrouting.ModeRoutingAuthoritative,
		safeheronRuntimeContext: ctx,
		companyFundRuntimeConfig: companyFundRuntimeConfig{
			StartBackgroundWorkers: true,
		},
	}

	finalizeSafeheronRouting(c)

	if c.FundRoutingStatusRefresher == nil {
		t.Fatal("routing-authoritative mode must wire the Safeheron provider status fallback")
	}
}
