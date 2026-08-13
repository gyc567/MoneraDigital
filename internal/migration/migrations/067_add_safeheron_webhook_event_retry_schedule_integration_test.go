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

const migration067PostgresIntegrationGate = "RUN_MIGRATION_067_POSTGRES_INTEGRATION"

func TestMigration067PostgresInstallsWebhookEventRetrySchedule(t *testing.T) {
	if os.Getenv(migration067PostgresIntegrationGate) != "1" {
		t.Skip("set RUN_MIGRATION_067_POSTGRES_INTEGRATION=1 to run isolated PostgreSQL coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when migration 067 PostgreSQL integration is enabled")
	}
	db, schema := newMigration067PostgresFixture(t, databaseURL)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			t.Errorf("rollback migration 067 transaction: %v", err)
		}
	})
	if _, err := tx.Exec(qualifyCompanyFundIntegrationSQL(migration067SchemaSQL, schema)); err != nil {
		t.Fatalf("apply migration 067: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`INSERT INTO ` + schema + `.safeheron_webhook_events
		(event_id,event_type,raw_payload,process_status,next_attempt_at)
		VALUES ('pending','AML_KYT_ALERT','{}','PENDING',now()+interval '1 minute')`); err != nil {
		t.Fatalf("pending deferred event rejected: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO ` + schema + `.safeheron_webhook_events
		(event_id,event_type,raw_payload,process_status,next_attempt_at)
		VALUES ('done','AML_KYT_ALERT','{}','DONE',now()+interval '1 minute')`); err == nil {
		t.Fatal("terminal event with next_attempt_at was accepted")
	}

	var indexCount int
	if err := db.QueryRow(`SELECT count(*) FROM pg_indexes WHERE schemaname=$1 AND indexname='idx_safeheron_webhook_events_pending_retry'`, schema).Scan(&indexCount); err != nil {
		t.Fatal(err)
	}
	if indexCount != 1 {
		t.Fatalf("pending retry index count = %d, want 1", indexCount)
	}
}

func newMigration067PostgresFixture(t *testing.T, databaseURL string) (*sql.DB, string) {
	t.Helper()
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	db := stdlib.OpenDB(*config)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close migration 067 database: %v", err)
		}
	})
	schema := fmt.Sprintf("migration_067_%d", time.Now().UnixNano())
	if _, err := db.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`); err != nil {
			t.Errorf("drop schema: %v", err)
		}
	})
	if _, err := db.Exec(`CREATE TABLE ` + schema + `.safeheron_webhook_events (
		id BIGSERIAL PRIMARY KEY,
		event_id VARCHAR(128) NOT NULL UNIQUE,
		event_type VARCHAR(64) NOT NULL,
		raw_payload JSONB NOT NULL,
		process_status VARCHAR(16) NOT NULL,
		received_at TIMESTAMP NOT NULL DEFAULT now()
	)`); err != nil {
		t.Fatal(err)
	}
	return db, schema
}
