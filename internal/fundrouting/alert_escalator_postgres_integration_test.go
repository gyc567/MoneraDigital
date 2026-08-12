package fundrouting

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func openRoutingSLAIntegrationDB(t *testing.T) (*sql.DB, context.Context) {
	t.Helper()
	if os.Getenv("RUN_FUND_ROUTING_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_FUND_ROUTING_POSTGRES_INTEGRATION=1 to run PostgreSQL routing coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	schema := fmt.Sprintf("fundrouting_sla_%x", time.Now().UnixNano())
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := db.ExecContext(ctx, `CREATE SCHEMA `+quotedSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SET search_path TO `+quotedSchema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `SET search_path TO public`)
		_, _ = db.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+quotedSchema+` CASCADE`)
	})

	for _, statement := range []string{
		`CREATE TABLE safeheron_transaction_routing_cases (
  id BIGINT PRIMARY KEY,
  safeheron_tx_key TEXT NOT NULL,
  raw_coin_key TEXT NOT NULL DEFAULT 'USDT_ERC20',
  network_family TEXT NOT NULL DEFAULT 'EVM',
  direction TEXT NOT NULL DEFAULT 'OUTFLOW',
  movement_kind TEXT NOT NULL DEFAULT 'PRINCIPAL',
  amount NUMERIC NOT NULL DEFAULT 1,
  normalized_source TEXT NOT NULL DEFAULT '0xsource',
  normalized_destination TEXT NOT NULL DEFAULT '0xdestination',
  effective_event_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  decision TEXT NOT NULL,
  reason_code TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
)`,
		`CREATE TABLE safeheron_transaction_routing_status_checks (
  safeheron_tx_key TEXT PRIMARY KEY,
  last_checked_at TIMESTAMPTZ,
  last_check_outcome TEXT,
  last_observed_status TEXT,
  last_provider_event_id TEXT,
  last_error_code TEXT
)`,
		`CREATE TABLE safeheron_webhook_events (
  id BIGINT PRIMARY KEY,
  event_id TEXT NOT NULL UNIQUE,
  event_type TEXT NOT NULL,
  raw_payload JSONB NOT NULL,
  received_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'UTC')
)`,
		`CREATE TABLE safeheron_transaction_routing_case_sources (
  id BIGINT PRIMARY KEY,
  case_id BIGINT NOT NULL,
  safeheron_webhook_event_id BIGINT NOT NULL,
  provider_status TEXT NOT NULL,
  provider_status_rank INTEGER NOT NULL,
  linked_at TIMESTAMPTZ NOT NULL
)`,
		`CREATE TABLE safeheron_transaction_routing_alerts (
  id BIGSERIAL PRIMARY KEY,
  case_id BIGINT NOT NULL,
  alert_type TEXT NOT NULL,
  transition_key TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'INFO',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (case_id,alert_type,transition_key)
)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	return db, ctx
}

func TestAlertEscalatorSQLPostgresNextDueSchedulesOnlyFutureOnChainThresholds(t *testing.T) {
	db, ctx := openRoutingSLAIntegrationDB(t)
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_cases
  (id,safeheron_tx_key,decision,reason_code,created_at)
VALUES (1,'tx-next-due','OPEN','STATUS_NOT_TERMINAL',now()-interval '2 hours')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_status_checks
  (safeheron_tx_key,last_check_outcome,last_observed_status)
VALUES ('tx-next-due','OBSERVED','SUBMITTED')`); err != nil {
		t.Fatal(err)
	}

	escalator, err := NewAlertEscalator(db)
	if err != nil {
		t.Fatal(err)
	}
	if due, err := escalator.NextDue(ctx); err != nil || !due.IsZero() {
		t.Fatalf("approval-stage NextDue() = %s, %v; want no business due time", due, err)
	}

	chainStartedAt := time.Now().UTC().Add(-30 * time.Minute).Round(time.Microsecond)
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_case_sources
  (id,case_id,safeheron_webhook_event_id,provider_status,provider_status_rank,linked_at)
VALUES (1,1,1,'BROADCASTING',20,$1)`, chainStartedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_status_checks
SET last_observed_status='BROADCASTING' WHERE safeheron_tx_key='tx-next-due'`); err != nil {
		t.Fatal(err)
	}
	due, err := escalator.NextDue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantDue := chainStartedAt.Add(time.Hour)
	if !due.Equal(wantDue) {
		t.Fatalf("on-chain NextDue() = %s, want %s", due, wantDue)
	}

	if _, err := db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_case_sources
SET linked_at=now()-interval '25 hours' WHERE case_id=1`); err != nil {
		t.Fatal(err)
	}
	if due, err := escalator.NextDue(ctx); err != nil || !due.IsZero() {
		t.Fatalf("expired NextDue() = %s, %v; want no past due time", due, err)
	}
}

func TestAlertEscalatorSQLPostgresKeepsApprovalQuietAndUsesCanonicalOnChainEvidence(t *testing.T) {
	db, ctx := openRoutingSLAIntegrationDB(t)
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_cases
  (id,safeheron_tx_key,decision,reason_code,created_at)
VALUES (1,'tx-escalation','OPEN','STATUS_NOT_TERMINAL',now()-interval '25 hours')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_webhook_events
  (id,event_id,event_type,raw_payload,received_at)
VALUES (1,'approval-event','TRANSACTION_CREATED',
        '{"eventDetail":{"transactionSubStatus":"PENDING_APPROVAL"}}',now()-interval '25 hours')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_case_sources
  (id,case_id,safeheron_webhook_event_id,provider_status,provider_status_rank,linked_at)
VALUES (1,1,1,'SUBMITTED',10,now()-interval '25 hours')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_status_checks
  (safeheron_tx_key,last_checked_at,last_check_outcome,last_observed_status,last_provider_event_id)
VALUES ('tx-escalation',now(),'OBSERVED','SUBMITTED','approval-event')`); err != nil {
		t.Fatal(err)
	}

	escalator, err := NewAlertEscalator(db)
	if err != nil {
		t.Fatal(err)
	}
	if worked, err := escalator.ProcessOne(ctx); err != nil || worked {
		t.Fatalf("approval-stage ProcessOne() = %v, %v; want quiet", worked, err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_webhook_events
  (id,event_id,event_type,raw_payload,received_at)
VALUES
  (2,'api-observed','TRANSACTION_STATUS_CHANGED',
   '{"eventDetail":{"transactionSubStatus":"API_OBSERVED","txHash":"0xapi"}}','2026-08-12 10:51:07.758845'),
  (3,'later-webhook','TRANSACTION_STATUS_CHANGED',
   '{"eventDetail":{"transactionSubStatus":"LATER_WEBHOOK","txHash":"0xlater"}}','2026-08-12 12:00:00')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_case_sources
  (id,case_id,safeheron_webhook_event_id,provider_status,provider_status_rank,linked_at)
VALUES
  (2,1,2,'BROADCASTING',20,now()-interval '25 hours'),
  (3,1,3,'BROADCASTING',20,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_status_checks
SET last_checked_at=now(),last_observed_status='BROADCASTING',last_provider_event_id='api-observed'
WHERE safeheron_tx_key='tx-escalation'`); err != nil {
		t.Fatal(err)
	}

	if worked, err := escalator.ProcessOne(ctx); err != nil || !worked {
		t.Fatalf("on-chain ProcessOne() = %v, %v; want one alert", worked, err)
	}
	if worked, err := escalator.ProcessOne(ctx); err != nil || worked {
		t.Fatalf("second ProcessOne() = %v, %v; want no lower-level catch-up", worked, err)
	}

	var count int
	var transitionKey, severity, subStatus, txHash string
	if err := db.QueryRowContext(ctx, `SELECT count(*),min(transition_key),min(severity),
       min(payload->>'transaction_sub_status'),min(payload->>'tx_hash')
FROM safeheron_transaction_routing_alerts
WHERE case_id=1 AND alert_type='SLA_ESCALATION'`).Scan(
		&count, &transitionKey, &severity, &subStatus, &txHash,
	); err != nil {
		t.Fatal(err)
	}
	if count != 1 || transitionKey != "sla:onchain:level:3" || severity != "CRITICAL" {
		t.Fatalf("alerts=%d transition=%q severity=%q", count, transitionKey, severity)
	}
	if subStatus != "API_OBSERVED" || txHash != "0xapi" {
		t.Fatalf("canonical evidence substatus=%q txHash=%q", subStatus, txHash)
	}
	var payload []byte
	if err := db.QueryRowContext(ctx, `SELECT payload
FROM safeheron_transaction_routing_alerts
WHERE case_id=1 AND alert_type='SLA_ESCALATION'`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	_, _, fields := routingAlertPresentation([]claimedDelivery{{
		Severity: "CRITICAL", AlertType: "SLA_ESCALATION", Payload: payload,
	}})
	if detail := fields["交易01"]; !strings.Contains(
		detail,
		"最后事件时间：2026-08-12 18:51:07 UTC+8",
	) {
		t.Fatalf("production TIMESTAMP was not rendered as UTC+8:\n%s", detail)
	}
}

func TestStatusRecoverySQLPostgresRequiresOnChainSLA(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		transition string
		wantRows   int64
	}{
		{name: "on-chain SLA", transition: "sla:onchain:level:1", wantRows: 1},
		{name: "legacy approval SLA", transition: "sla:pending:level:1", wantRows: 0},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db, ctx := openRoutingSLAIntegrationDB(t)
			if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_cases
  (id,safeheron_tx_key,decision,reason_code,created_at)
VALUES (1,'tx-recovery','OPEN','STATUS_NOT_TERMINAL',now()-interval '2 hours')`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_webhook_events
  (id,event_id,event_type,raw_payload,received_at)
VALUES
  (1,'broadcast-event','TRANSACTION_STATUS_CHANGED','{}',now()-interval '2 hours'),
  (2,'recovery-event','TRANSACTION_STATUS_CHANGED','{}',now())`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_case_sources
  (id,case_id,safeheron_webhook_event_id,provider_status,provider_status_rank,linked_at)
VALUES
  (1,1,1,'BROADCASTING',20,now()-interval '2 hours'),
  (2,1,2,'COMPLETED',100,now())`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_status_checks
  (safeheron_tx_key,last_checked_at,last_check_outcome,last_observed_status,last_provider_event_id)
VALUES ('tx-recovery',now(),'OBSERVED','COMPLETED','recovery-event')`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_alerts
  (case_id,alert_type,transition_key,severity,payload)
VALUES (1,'SLA_ESCALATION',$1,'WARN','{"reason_code":"STATUS_NOT_TERMINAL"}')`, testCase.transition); err != nil {
				t.Fatal(err)
			}

			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			result, err := tx.ExecContext(ctx, statusRecoveryAlertSQL(),
				1, 2, "COMPANY", "COMPANY_ADDRESS_MATCH", "tx-recovery", "USDT_ERC20", "EVM",
				"OUTFLOW", "220", "0xsource", "0xdestination", "COMPLETED", "0xhash", "PRINCIPAL", "",
			)
			if err != nil {
				_ = tx.Rollback()
				t.Fatal(err)
			}
			rows, err := result.RowsAffected()
			if err != nil {
				_ = tx.Rollback()
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if rows != testCase.wantRows {
				t.Fatalf("recovery rows=%d, want %d", rows, testCase.wantRows)
			}
		})
	}
}
