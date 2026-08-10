package migrations

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

const migration064PostgresIntegrationGate = "RUN_MIGRATION_064_POSTGRES_INTEGRATION"

func TestMigration064PostgresInstallsStatusChecksAndRekeysExistingSLAAlerts(t *testing.T) {
	if os.Getenv(migration064PostgresIntegrationGate) != "1" {
		t.Skip("set RUN_MIGRATION_064_POSTGRES_INTEGRATION=1 to run isolated PostgreSQL coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when migration 064 PostgreSQL integration is enabled")
	}
	db, schema := newMigration064PostgresFixture(t, databaseURL)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(qualifyCompanyFundIntegrationSQL(migration064SchemaSQL, schema)); err != nil {
		t.Fatalf("apply migration 064: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query(`SELECT transition_key FROM ` + schema + `.safeheron_transaction_routing_alerts ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var transitions []string
	for rows.Next() {
		var transition string
		if err := rows.Scan(&transition); err != nil {
			t.Fatal(err)
		}
		transitions = append(transitions, transition)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(transitions) != 2 || transitions[0] != "sla:pending:level:1" || transitions[1] != "sla:open:level:2" {
		t.Fatalf("transition keys = %#v", transitions)
	}

	insert := `INSERT INTO ` + schema + `.safeheron_transaction_routing_status_checks
  (safeheron_tx_key,first_seen_at,next_check_at,last_checked_at,last_check_outcome,last_observed_status,last_provider_event_id)
VALUES ($1,now(),now(),now(),'OBSERVED','SUBMITTED',$2)`
	if _, err := db.Exec(insert, "tx-valid", "safeheron-routing-status:v1:event"); err != nil {
		t.Fatalf("valid observed check rejected: %v", err)
	}
	if _, err := db.Exec(`UPDATE ` + schema + `.safeheron_transaction_routing_status_checks
SET last_check_outcome='ERROR',last_error_code='PROVIDER_LOOKUP_FAILED'
WHERE safeheron_tx_key='tx-valid'`); err == nil {
		t.Fatal("ERROR outcome with provider event identity was accepted")
	}
}

func newMigration064PostgresFixture(t *testing.T, databaseURL string) (*sql.DB, string) {
	t.Helper()
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	db := stdlib.OpenDB(*config)
	t.Cleanup(func() { _ = db.Close() })
	schema := fmt.Sprintf("migration_064_%d", time.Now().UnixNano())
	if _, err := db.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`); err != nil {
			t.Errorf("drop schema: %v", err)
		}
	})
	fixtureSQL := `CREATE TABLE ` + schema + `.safeheron_transaction_routing_alerts (
  id BIGSERIAL PRIMARY KEY,
  alert_type VARCHAR(64) NOT NULL,
  transition_key VARCHAR(128) NOT NULL,
  payload JSONB NOT NULL
);
INSERT INTO ` + schema + `.safeheron_transaction_routing_alerts (alert_type,transition_key,payload) VALUES
  ('SLA_ESCALATION','sla:level:1','{"reason_code":"STATUS_NOT_TERMINAL","level":1}'),
  ('SLA_ESCALATION','sla:level:2','{"reason_code":"COMPANY_DISABLED","level":2}');`
	if _, err := db.Exec(fixtureSQL); err != nil {
		t.Fatal(err)
	}
	return db, schema
}
