package fundrouting

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const routingAlertQueuePostgresGate = "RUN_FUND_ROUTING_QUEUE_POSTGRES_INTEGRATION"

func TestRoutingAlertQueuePostgresClaimsStableLarkBatchAndSkipsLockedRows(t *testing.T) {
	db, queue := newRoutingAlertQueuePostgresFixture(t)
	ctx := context.Background()
	fingerprint := strings.Repeat("a", 64)
	createdAt := time.Date(2026, 8, 10, 4, 26, 3, 0, time.UTC)
	due := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	insertRoutingAlertQueueFixture(t, db, 10, 110, 1010, "LARK", fingerprint, "PENDING", 0, due, createdAt)
	insertRoutingAlertQueueFixture(t, db, 11, 111, 1011, "LARK", fingerprint, "PENDING", 0, due, createdAt)

	batch, err := queue.Claim(ctx, "lark-worker", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 2 || batch[0].ID != 10 || batch[1].ID != 11 {
		t.Fatalf("Lark batch=%#v, want stable delivery order [10, 11]", batch)
	}
	var leased, attempts int
	if err := db.QueryRow(`SELECT count(*) FROM safeheron_transaction_routing_alert_deliveries
WHERE id IN (10,11) AND status='DISPATCHING' AND lease_owner='lark-worker'`).Scan(&leased); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM safeheron_transaction_routing_alert_delivery_attempts
WHERE delivery_id IN (10,11) AND attempt_kind='AUTO' AND outcome='IN_PROGRESS'`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if leased != 2 || attempts != 2 {
		t.Fatalf("leased=%d attempts=%d, want 2/2", leased, attempts)
	}

	insertRoutingAlertQueueFixture(t, db, 20, 120, 1020, "EMAIL", strings.Repeat("b", 64), "PENDING", 0, time.Time{}, createdAt)
	insertRoutingAlertQueueFixture(t, db, 21, 121, 1021, "EMAIL", strings.Repeat("c", 64), "PENDING", 0, time.Time{}, createdAt)
	locker, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = locker.Rollback() })
	var lockedID int64
	if err := locker.QueryRow(`SELECT id FROM safeheron_transaction_routing_alert_deliveries
WHERE id=20 FOR UPDATE`).Scan(&lockedID); err != nil {
		t.Fatal(err)
	}
	claimCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	batch, err = queue.Claim(claimCtx, "skip-locked-worker", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 1 || batch[0].ID != 21 {
		t.Fatalf("SKIP LOCKED batch=%#v, want delivery 21", batch)
	}
}

func TestRoutingAlertQueuePostgresPreservesManualReplayAttempt(t *testing.T) {
	db, queue := newRoutingAlertQueuePostgresFixture(t)
	createdAt := time.Date(2026, 8, 10, 4, 26, 3, 0, time.UTC)
	due := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	insertRoutingAlertQueueFixture(
		t,
		db,
		30,
		130,
		1030,
		"LARK",
		strings.Repeat("d", 64),
		"FAILED_DEFINITE",
		2,
		due,
		createdAt,
	)
	if _, err := db.Exec(`INSERT INTO safeheron_transaction_routing_alert_delivery_attempts
  (id,delivery_id,attempt_number,attempt_kind,outcome,actor_admin_user_id,reason,idempotency_key)
VALUES (230,30,3,'MANUAL_REPLAY','IN_PROGRESS',42,'operator-approved replay','manual-replay-30')`); err != nil {
		t.Fatal(err)
	}

	batch, err := queue.Claim(context.Background(), "manual-worker", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 1 || batch[0].ID != 30 || batch[0].AttemptID != 230 ||
		batch[0].Attempt != 3 || batch[0].AutomaticAttemptCount != 2 {
		t.Fatalf("manual replay batch=%#v", batch)
	}
	var attempts int
	if err := db.QueryRow(`SELECT count(*) FROM safeheron_transaction_routing_alert_delivery_attempts
WHERE delivery_id=30`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("manual replay created %d attempts, want the existing one only", attempts)
	}
}

func TestRoutingAlertQueuePostgresClaimsFailedDeliveryAsAutomaticRetry(t *testing.T) {
	db, queue := newRoutingAlertQueuePostgresFixture(t)
	createdAt := time.Date(2026, 8, 10, 4, 26, 3, 0, time.UTC)
	due := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	insertRoutingAlertQueueFixture(
		t,
		db,
		40,
		140,
		1040,
		"LARK",
		strings.Repeat("e", 64),
		"FAILED_DEFINITE",
		1,
		due,
		createdAt,
	)
	if _, err := db.Exec(`INSERT INTO safeheron_transaction_routing_alert_delivery_attempts
  (id,delivery_id,attempt_number,attempt_kind,outcome,finished_at)
VALUES (240,40,1,'AUTO','DEFINITELY_NOT_SENT',$1)`, createdAt); err != nil {
		t.Fatal(err)
	}

	batch, err := queue.Claim(context.Background(), "retry-worker", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 1 || batch[0].ID != 40 || batch[0].Attempt != 2 ||
		batch[0].AutomaticAttemptCount != 2 {
		t.Fatalf("automatic retry batch=%#v", batch)
	}
	var attempts, inProgress int
	if err := db.QueryRow(`SELECT count(*),count(*) FILTER (WHERE attempt_number=2 AND attempt_kind='AUTO' AND outcome='IN_PROGRESS')
FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id=40`).Scan(&attempts, &inProgress); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || inProgress != 1 {
		t.Fatalf("automatic retry attempts=%d in_progress=%d, want 2/1", attempts, inProgress)
	}
}

func newRoutingAlertQueuePostgresFixture(t *testing.T) (*sql.DB, *postgresRoutingAlertQueue) {
	t.Helper()
	if os.Getenv(routingAlertQueuePostgresGate) != "1" {
		t.Skip("set " + routingAlertQueuePostgresGate + "=1 to run PostgreSQL queue coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required")
	}
	adminDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adminDB.Close() })
	schema := fmt.Sprintf("routing_alert_queue_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`)
	})
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = db.Close() })
	if err := createRoutingAlertQueuePostgresSchema(db); err != nil {
		t.Fatal(err)
	}
	queue, err := newPostgresRoutingAlertQueue(db)
	if err != nil {
		t.Fatal(err)
	}
	return db, queue
}

func createRoutingAlertQueuePostgresSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE safeheron_transaction_routing_cases (id BIGINT PRIMARY KEY)`,
		`CREATE TABLE safeheron_transaction_routing_alerts (
  id BIGINT PRIMARY KEY,
  case_id BIGINT NOT NULL REFERENCES safeheron_transaction_routing_cases(id) ON DELETE RESTRICT,
  alert_type VARCHAR(32) NOT NULL CHECK (alert_type IN ('OPEN','ACTION_DEAD','SLA_ESCALATION','RECOVERY_SUMMARY')),
  transition_key VARCHAR(256) NOT NULL,
  severity VARCHAR(16) NOT NULL CHECK (severity IN ('INFO','WARN','ERROR','CRITICAL')),
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  UNIQUE (case_id,alert_type,transition_key)
)`,
		`CREATE TABLE safeheron_transaction_routing_alert_deliveries (
  id BIGINT PRIMARY KEY,
  alert_id BIGINT NOT NULL REFERENCES safeheron_transaction_routing_alerts(id) ON DELETE RESTRICT,
  sink_kind VARCHAR(16) NOT NULL CHECK (sink_kind IN ('LARK','EMAIL')),
  recipient_fingerprint VARCHAR(64) NOT NULL CHECK (recipient_fingerprint ~ '^[0-9a-f]{64}$'),
  status VARCHAR(24) NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING','DISPATCHING','SENT','FAILED_DEFINITE','AMBIGUOUS')),
  automatic_attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (automatic_attempt_count >= 0),
  next_attempt_at TIMESTAMPTZ,
  lease_owner VARCHAR(128),
  lease_expires_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (alert_id,sink_kind,recipient_fingerprint),
  CHECK (
    (status='DISPATCHING' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
    OR (status<>'DISPATCHING' AND lease_owner IS NULL AND lease_expires_at IS NULL)
  ),
  CHECK (
    (status IN ('SENT','AMBIGUOUS') AND completed_at IS NOT NULL)
    OR (status IN ('PENDING','DISPATCHING','FAILED_DEFINITE') AND completed_at IS NULL)
  )
)`,
		`CREATE TABLE safeheron_transaction_routing_alert_delivery_attempts (
  id BIGSERIAL PRIMARY KEY,
  delivery_id BIGINT NOT NULL REFERENCES safeheron_transaction_routing_alert_deliveries(id) ON DELETE RESTRICT,
  attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
  attempt_kind VARCHAR(24) NOT NULL CHECK (attempt_kind IN ('AUTO','MANUAL_REPLAY')),
  outcome VARCHAR(32) NOT NULL CHECK (outcome IN ('IN_PROGRESS','SENT','DEFINITELY_NOT_SENT','DELIVERY_UNKNOWN')),
  actor_admin_user_id BIGINT,
  reason TEXT,
  idempotency_key VARCHAR(128),
  finished_at TIMESTAMPTZ,
  UNIQUE (delivery_id,attempt_number),
  CHECK (
    (attempt_kind='AUTO' AND actor_admin_user_id IS NULL AND reason IS NULL AND idempotency_key IS NULL)
    OR (attempt_kind='MANUAL_REPLAY' AND actor_admin_user_id IS NOT NULL AND reason IS NOT NULL
      AND btrim(reason)<>'' AND idempotency_key IS NOT NULL)
  ),
  CHECK (
    (outcome='IN_PROGRESS' AND finished_at IS NULL)
    OR (outcome<>'IN_PROGRESS' AND finished_at IS NOT NULL)
  )
)`,
		`CREATE UNIQUE INDEX idx_routing_alert_manual_replay_idempotency
ON safeheron_transaction_routing_alert_delivery_attempts (delivery_id,idempotency_key)
WHERE attempt_kind='MANUAL_REPLAY'`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func insertRoutingAlertQueueFixture(
	t *testing.T,
	db *sql.DB,
	deliveryID int64,
	alertID int64,
	caseID int64,
	sinkKind string,
	fingerprint string,
	status string,
	automaticAttempts int,
	nextAttempt time.Time,
	createdAt time.Time,
) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO safeheron_transaction_routing_cases (id) VALUES ($1)`, caseID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO safeheron_transaction_routing_alerts
  (id,case_id,severity,alert_type,transition_key,payload,created_at)
VALUES ($1,$2,'WARN','SLA_ESCALATION',$3,'{"case_id":1}'::jsonb,$4)`,
		alertID, caseID, fmt.Sprintf("test-transition-%d", alertID), createdAt); err != nil {
		t.Fatal(err)
	}
	var due any
	if !nextAttempt.IsZero() {
		due = nextAttempt
	}
	if _, err := db.Exec(`INSERT INTO safeheron_transaction_routing_alert_deliveries
  (id,alert_id,sink_kind,recipient_fingerprint,status,automatic_attempt_count,next_attempt_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)`,
		deliveryID,
		alertID,
		sinkKind,
		fingerprint,
		status,
		automaticAttempts,
		due,
		createdAt,
	); err != nil {
		t.Fatal(err)
	}
}
