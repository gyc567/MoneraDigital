package companyfund

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/shopspring/decimal"

	"monera-digital/internal/safeheron"
)

const safeheronAssetRecognitionRepairPostgresGate = "RUN_COMPANY_FUND_SAFEHERON_ASSET_REPAIR_POSTGRES_INTEGRATION"

func TestSafeheronAssetRecognitionRepairPostgresIntegration(t *testing.T) {
	if os.Getenv(safeheronAssetRecognitionRepairPostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_SAFEHERON_ASSET_REPAIR_POSTGRES_INTEGRATION=1 to run the isolated recognition repair contract")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when the recognition repair integration contract is enabled")
	}
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { _ = admin.Close() })
	schema := fmt.Sprintf("safeheron_asset_recognition_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(context.Background(), `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`) })
	fixtureConfig := config.Copy()
	fixtureConfig.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*fixtureConfig)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(context.Background(), `CREATE TABLE company_fund_transactions (
		id BIGINT PRIMARY KEY,
		channel TEXT NOT NULL,
		movement_key TEXT NOT NULL,
		amount NUMERIC NOT NULL,
		currency TEXT NOT NULL,
		chain_code TEXT,
		provider_asset_key TEXT,
		asset_contract TEXT,
		is_unrecognized_asset BOOLEAN NOT NULL,
		finance_category_level1_id BIGINT,
		finance_category_level2_id BIGINT,
		is_operating_income_expense BOOLEAN,
		usd_value NUMERIC,
		current_valuation_history_id BIGINT,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO company_fund_transactions (
		id, channel, movement_key, amount, currency, provider_asset_key, is_unrecognized_asset,
		finance_category_level1_id, finance_category_level2_id, is_operating_income_expense,
		usd_value, current_valuation_history_id
	) VALUES (385, 'SAFEHERON', 'immutable-movement', 3600, 'USDT_ERC20', 'USDT_ERC20', true,
		11, 22, true, 3599.99, 77)`); err != nil {
		t.Fatal(err)
	}

	catalog, err := NewSafeheronCoinCatalog(&fakeSafeheronCoinLister{coins: []safeheron.Coin{{
		CoinKey: "USDT_ERC20", Symbol: "USDT", BlockChain: "ETHEREUM", BlockchainType: "EVM",
		TokenIdentifier: "0xDaC17F",
	}}}, SafeheronCoinCatalogConfig{})
	if err != nil || catalog.Refresh(t.Context()) != nil {
		t.Fatalf("catalog setup: %v", err)
	}
	repairer, err := NewSafeheronAssetRecognitionRepairer(NewDBRepository(db), catalog)
	if err != nil {
		t.Fatal(err)
	}
	result, err := repairer.Sweep(t.Context(), 10)
	if err != nil || result.Repaired != 1 {
		t.Fatalf("repair result = %#v, %v", result, err)
	}

	var (
		movementKey, amountText, currency, chainCode, providerAssetKey, contract, usdText string
		unrecognized, operating                                                           bool
		level1, level2, valuationHistory                                                  int64
	)
	if err := db.QueryRowContext(context.Background(), `SELECT movement_key, amount::TEXT, currency,
		chain_code, provider_asset_key, asset_contract, is_unrecognized_asset,
		finance_category_level1_id, finance_category_level2_id, is_operating_income_expense,
		usd_value::TEXT, current_valuation_history_id
		FROM company_fund_transactions WHERE id=385`).Scan(
		&movementKey, &amountText, &currency, &chainCode, &providerAssetKey, &contract, &unrecognized,
		&level1, &level2, &operating, &usdText, &valuationHistory,
	); err != nil {
		t.Fatal(err)
	}
	amount, amountErr := decimal.NewFromString(amountText)
	usdValue, usdErr := decimal.NewFromString(usdText)
	if amountErr != nil || usdErr != nil || movementKey != "immutable-movement" || !amount.Equal(decimal.NewFromInt(3600)) ||
		currency != "USDT" || chainCode != "ETHEREUM" || providerAssetKey != "USDT_ERC20" ||
		contract != "0xdac17f" || unrecognized || level1 != 11 || level2 != 22 || !operating ||
		!usdValue.Equal(decimal.RequireFromString("3599.99")) || valuationHistory != 77 {
		t.Fatalf("repaired row changed protected data: movement=%q amount=%q currency=%q chain=%q provider=%q contract=%q unrecognized=%t categories=%d/%d operating=%t usd=%q history=%d",
			movementKey, amountText, currency, chainCode, providerAssetKey, contract, unrecognized,
			level1, level2, operating, usdText, valuationHistory)
	}

	second, err := repairer.Sweep(t.Context(), 10)
	if err != nil || second.Scanned != 0 || second.Repaired != 0 {
		t.Fatalf("idempotent second repair = %#v, %v", second, err)
	}
}
