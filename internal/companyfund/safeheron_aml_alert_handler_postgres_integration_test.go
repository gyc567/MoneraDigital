package companyfund

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const safeheronAMLAlertPostgresGate = "RUN_COMPANY_FUND_SAFEHERON_AML_INTEGRATION"

func TestSafeheronAMLAlertHandlerPostgresRoutingOwnership(t *testing.T) {
	if os.Getenv(safeheronAMLAlertPostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_SAFEHERON_AML_INTEGRATION=1 to run isolated PostgreSQL AML routing coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when Safeheron AML integration tests are enabled")
	}

	db := newSafeheronAMLAlertPostgresFixture(t, databaseURL)
	insertSafeheronAMLAlertRoutingFixture(t, db)
	handler := NewSafeheronAMLAlertHandler(db)

	t.Run("open routing case defers instead of becoming customer orphan", func(t *testing.T) {
		result, err := handler.HandleCompanyFundAMLAlert(context.Background(), SafeheronAMLAlertInput{
			TransactionKey: "tx-open",
			ScreeningState: "TRIGGERED",
			RiskLevel:      "LOW",
		})
		if err != nil || result != SafeheronAMLAlertDeferred {
			t.Fatalf("HandleCompanyFundAMLAlert() = %q, %v; want DEFERRED", result, err)
		}
	})

	t.Run("deferred alert applies after the same routing case becomes company owned", func(t *testing.T) {
		if _, err := db.ExecContext(context.Background(), `
INSERT INTO company_fund_transactions (
  id, channel, provider_transaction_id, aml_screening_state, aml_risk_level
) VALUES (102, 'SAFEHERON', 'tx-open', 'NOT_SCREENED', 'UNKNOWN');
UPDATE safeheron_transaction_routing_cases
SET decision = 'COMPANY',
    requires_company_projection = true,
    company_fund_transaction_id = 102
WHERE safeheron_tx_key = 'tx-open'`); err != nil {
			t.Fatalf("complete company routing fixture: %v", err)
		}

		result, err := handler.HandleCompanyFundAMLAlert(context.Background(), SafeheronAMLAlertInput{
			TransactionKey: "tx-open",
			ScreeningState: "TRIGGERED",
			RiskLevel:      "LOW",
		})
		if err != nil || result != SafeheronAMLAlertApplied {
			t.Fatalf("HandleCompanyFundAMLAlert() = %q, %v; want APPLIED", result, err)
		}
	})

	t.Run("projected company routing applies AML result", func(t *testing.T) {
		result, err := handler.HandleCompanyFundAMLAlert(context.Background(), SafeheronAMLAlertInput{
			TransactionKey: "tx-company",
			ScreeningState: "TRIGGERED",
			RiskLevel:      "LOW",
		})
		if err != nil || result != SafeheronAMLAlertApplied {
			t.Fatalf("HandleCompanyFundAMLAlert() = %q, %v; want APPLIED", result, err)
		}
	})

	t.Run("customer routing defers until its deposit projection exists", func(t *testing.T) {
		result, err := handler.HandleCompanyFundAMLAlert(context.Background(), SafeheronAMLAlertInput{
			TransactionKey: "tx-customer-pending",
			ScreeningState: "TRIGGERED",
			RiskLevel:      "LOW",
		})
		if err != nil || result != SafeheronAMLAlertDeferred {
			t.Fatalf("HandleCompanyFundAMLAlert() = %q, %v; want DEFERRED", result, err)
		}
	})

	t.Run("customer routing stays in deposit pipeline", func(t *testing.T) {
		result, err := handler.HandleCompanyFundAMLAlert(context.Background(), SafeheronAMLAlertInput{
			TransactionKey: "tx-customer",
			ScreeningState: "TRIGGERED",
			RiskLevel:      "LOW",
		})
		if err != nil || result != SafeheronAMLAlertNotCompany {
			t.Fatalf("HandleCompanyFundAMLAlert() = %q, %v; want NOT_COMPANY", result, err)
		}
	})
}

func newSafeheronAMLAlertPostgresFixture(t *testing.T, databaseURL string) *sql.DB {
	t.Helper()
	adminConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	adminDB := stdlib.OpenDB(*adminConfig)
	t.Cleanup(func() {
		if err := adminDB.Close(); err != nil {
			t.Errorf("close PostgreSQL admin connection: %v", err)
		}
	})

	schema := fmt.Sprintf("safeheron_aml_alert_%d", time.Now().UnixNano())
	if _, err := adminDB.ExecContext(context.Background(), `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := adminDB.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`); err != nil {
			t.Errorf("drop isolated schema %s: %v", schema, err)
		}
	})

	fixtureConfig := adminConfig.Copy()
	if fixtureConfig.RuntimeParams == nil {
		fixtureConfig.RuntimeParams = make(map[string]string)
	}
	fixtureConfig.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*fixtureConfig)
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close PostgreSQL fixture connection: %v", err)
		}
	})

	if _, err := db.ExecContext(context.Background(), `
CREATE TABLE safeheron_transaction_routing_cases (
  id BIGSERIAL PRIMARY KEY,
  safeheron_tx_key VARCHAR(128) NOT NULL,
  decision VARCHAR(24) NOT NULL,
  requires_customer_projection BOOLEAN NOT NULL DEFAULT false,
  requires_company_projection BOOLEAN NOT NULL DEFAULT false,
  deposit_id BIGINT,
  company_fund_transaction_id BIGINT
);
CREATE TABLE company_fund_transactions (
  id BIGINT PRIMARY KEY,
  channel VARCHAR(32) NOT NULL,
  provider_transaction_id VARCHAR(256) NOT NULL,
  aml_screening_state VARCHAR(32),
  aml_risk_level VARCHAR(32),
  last_synced_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`); err != nil {
		t.Fatalf("create Safeheron AML fixture: %v", err)
	}
	return db
}

func insertSafeheronAMLAlertRoutingFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO safeheron_transaction_routing_cases (
  safeheron_tx_key, decision, requires_customer_projection,
  requires_company_projection, deposit_id, company_fund_transaction_id
) VALUES
  ('tx-open', 'OPEN', false, false, NULL, NULL),
  ('tx-company', 'COMPANY', false, true, NULL, 101),
  ('tx-customer-pending', 'CUSTOMER', true, false, NULL, NULL),
  ('tx-customer', 'CUSTOMER', true, false, 201, NULL);
INSERT INTO company_fund_transactions (
  id, channel, provider_transaction_id, aml_screening_state, aml_risk_level
) VALUES (101, 'SAFEHERON', 'tx-company', 'NOT_SCREENED', 'UNKNOWN');`); err != nil {
		t.Fatalf("insert Safeheron AML routing fixture: %v", err)
	}
}
