package fundrouting

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"monera-digital/internal/companyfund"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestStatusRefresherPostgresIngestsTerminalLookupThroughCanonicalRoutingInbox(t *testing.T) {
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
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	txKey := "routing-status-refresh-" + suffix
	initial := routingSnapshot()
	initial.TxKey = txKey
	initial.TxHash = ""
	initial.TransactionStatus = "SUBMITTED"
	initial.CreateTime = time.Now().Add(-10 * time.Minute).UnixMilli()
	initialPayload, _ := json.Marshal(map[string]any{
		"eventType": "TRANSACTION_CREATED", "eventDetail": initial,
	})
	initialDigest := fmt.Sprintf("%064x", time.Now().UnixNano())
	var initialWebhookID, caseID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO safeheron_webhook_events
  (event_id,event_type,safeheron_tx_key,raw_payload,payload_digest,process_status)
VALUES ($1,'TRANSACTION_CREATED',$2,$3::jsonb,$4,'PENDING') RETURNING id`,
		"routing-status-created-"+suffix, txKey, initialPayload, initialDigest,
	).Scan(&initialWebhookID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_status_checks WHERE safeheron_tx_key=$1`, txKey)
		if caseID > 0 {
			_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id IN (
              SELECT delivery.id FROM safeheron_transaction_routing_alert_deliveries delivery
              JOIN safeheron_transaction_routing_alerts alert ON alert.id=delivery.alert_id
              WHERE alert.case_id=$1)`, caseID)
			_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_alert_deliveries WHERE alert_id IN (
              SELECT id FROM safeheron_transaction_routing_alerts WHERE case_id=$1)`, caseID)
			_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_alerts WHERE case_id=$1`, caseID)
			_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_case_sources WHERE case_id=$1`, caseID)
			_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_cases WHERE id=$1`, caseID)
		}
		_, _ = db.Exec(`DELETE FROM company_fund_safeheron_raw_event_exclusions
          WHERE safeheron_webhook_event_id IN (SELECT id FROM safeheron_webhook_events WHERE safeheron_tx_key=$1)`, txKey)
		_, _ = db.Exec(`DELETE FROM safeheron_webhook_events WHERE safeheron_tx_key=$1`, txKey)
	})

	results, err := NewRepository(db).RouteVerifiedEvent(ctx, VerifiedEventInput{
		WebhookEventID: initialWebhookID, EventType: "TRANSACTION_CREATED",
		PayloadDigest: initialDigest, NetworkFamily: "EVM", Snapshot: initial,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Decision.Reason != ReasonStatusNotTerminal {
		t.Fatalf("initial route = %#v, want one STATUS_NOT_TERMINAL case", results)
	}
	caseID = results[0].CaseID
	if _, err := db.Exec(`UPDATE safeheron_transaction_routing_cases
SET created_at=now()-interval '10 minutes' WHERE id=$1`, caseID); err != nil {
		t.Fatal(err)
	}

	terminal := initial
	terminal.TxHash = "0x" + suffix
	terminal.TransactionStatus = "COMPLETED"
	terminalRaw, _ := json.Marshal(terminal)
	terminal.RawPayload = terminalRaw
	lookup := &routingStatusLookupStub{snapshot: &terminal}
	store, _ := NewPostgresRoutingStatusCheckStore(db)
	ingester, _ := NewHistoryInboxIngester(db)
	refresher, err := NewStatusRefresher(store, lookup, ingester, StatusRefresherConfig{
		WorkerID: "postgres-integration", Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}

	worked, err := refresher.ProcessOne(ctx)
	if err != nil || !worked {
		t.Fatalf("ProcessOne() = %v, %v", worked, err)
	}
	var completedAt sql.NullTime
	var status, eventType, processStatus string
	if err := db.QueryRow(`SELECT completed_at,last_observed_status
FROM safeheron_transaction_routing_status_checks WHERE safeheron_tx_key=$1`, txKey).
		Scan(&completedAt, &status); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT event_type,process_status FROM safeheron_webhook_events
WHERE event_id=$1`, routingStatusSnapshotEventID(terminal)).Scan(&eventType, &processStatus); err != nil {
		t.Fatal(err)
	}
	if !completedAt.Valid || status != "COMPLETED" || eventType != "TRANSACTION_STATUS_CHANGED" || processStatus != "PENDING" {
		t.Fatalf("check completed=%v status=%s event=%s process=%s", completedAt.Valid, status, eventType, processStatus)
	}
}

func TestAlertEscalatorPostgresUsesTheObservedAPISourceAfterALaterSameStatusWebhook(t *testing.T) {
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
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	txKey := "routing-status-source-" + suffix
	var caseID int64
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id IN (
          SELECT delivery.id FROM safeheron_transaction_routing_alert_deliveries delivery
          JOIN safeheron_transaction_routing_alerts alert ON alert.id=delivery.alert_id WHERE alert.case_id=$1)`, caseID)
		_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_alert_deliveries WHERE alert_id IN (
          SELECT id FROM safeheron_transaction_routing_alerts WHERE case_id=$1)`, caseID)
		_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_alerts WHERE case_id=$1`, caseID)
		_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_case_sources WHERE case_id=$1`, caseID)
		_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_status_checks WHERE safeheron_tx_key=$1`, txKey)
		_, _ = db.Exec(`DELETE FROM safeheron_transaction_routing_cases WHERE id=$1`, caseID)
		_, _ = db.Exec(`DELETE FROM company_fund_safeheron_raw_event_exclusions
          WHERE safeheron_webhook_event_id IN (SELECT id FROM safeheron_webhook_events WHERE safeheron_tx_key=$1)`, txKey)
		_, _ = db.Exec(`DELETE FROM safeheron_webhook_events WHERE safeheron_tx_key=$1`, txKey)
	})

	initial := routingSnapshot()
	initial.TxKey = txKey
	initial.TxHash = ""
	initial.TransactionStatus = "SUBMITTED"
	initial.TransactionSubStatus = "INITIAL_WEBHOOK"
	initial.CreateTime = time.Now().Add(-25 * time.Hour).UnixMilli()
	initialBody, _ := json.Marshal(initial)
	initial.RawPayload = initialBody
	initialEnvelope, _ := json.Marshal(map[string]any{"eventType": "TRANSACTION_CREATED", "eventDetail": initial})
	initialDigest := fmt.Sprintf("%064x", time.Now().UnixNano())
	var initialWebhookID int64
	if err := db.QueryRow(`INSERT INTO safeheron_webhook_events
  (event_id,event_type,safeheron_tx_key,raw_payload,payload_digest,process_status)
VALUES ($1,'TRANSACTION_CREATED',$2,$3::jsonb,$4,'PENDING') RETURNING id`,
		"routing-status-source-initial-"+suffix, txKey, initialEnvelope, initialDigest,
	).Scan(&initialWebhookID); err != nil {
		t.Fatal(err)
	}
	repository := NewRepository(db)
	results, err := repository.RouteVerifiedEvent(ctx, VerifiedEventInput{
		WebhookEventID: initialWebhookID, EventType: "TRANSACTION_CREATED", PayloadDigest: initialDigest,
		NetworkFamily: "EVM", Snapshot: initial,
	})
	if err != nil || len(results) != 1 || results[0].Decision.Reason != ReasonStatusNotTerminal {
		t.Fatalf("initial route = %#v, %v", results, err)
	}
	caseID = results[0].CaseID
	if _, err := db.Exec(`UPDATE safeheron_transaction_routing_cases
SET created_at=now()-interval '25 hours' WHERE id=$1`, caseID); err != nil {
		t.Fatal(err)
	}

	observed := initial
	observed.TxHash = "0x" + suffix
	observed.TransactionStatus = "BROADCASTING"
	observed.TransactionSubStatus = "API_OBSERVED"
	observedBody, _ := json.Marshal(observed)
	observed.RawPayload = observedBody
	providerEventID := routingStatusSnapshotEventID(observed)
	ingester, _ := NewHistoryInboxIngester(db)
	ingested, err := ingester.Ingest(ctx, companyfund.OwnedProviderPayloadInput{
		Channel: companyfund.ChannelSafeheron, ProviderEventID: providerEventID,
		EventType: companyfund.SafeheronTransactionHistorySnapshotEventType, Body: observedBody,
	})
	if err != nil {
		t.Fatal(err)
	}
	var observedDigest string
	if err := db.QueryRow(`SELECT payload_digest FROM safeheron_webhook_events WHERE id=$1`, ingested.ID).Scan(&observedDigest); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.RouteVerifiedEvent(ctx, VerifiedEventInput{
		WebhookEventID: ingested.ID, EventType: "TRANSACTION_STATUS_CHANGED", PayloadDigest: observedDigest,
		NetworkFamily: "EVM", Snapshot: observed,
	}); err != nil {
		t.Fatal(err)
	}

	later := observed
	later.TransactionSubStatus = "LATER_WEBHOOK"
	laterBody, _ := json.Marshal(later)
	later.RawPayload = laterBody
	laterEnvelope, _ := json.Marshal(map[string]any{"eventType": "TRANSACTION_STATUS_CHANGED", "eventDetail": later})
	laterDigest := fmt.Sprintf("%064x", time.Now().UnixNano()+1)
	var laterWebhookID int64
	if err := db.QueryRow(`INSERT INTO safeheron_webhook_events
  (event_id,event_type,safeheron_tx_key,raw_payload,payload_digest,process_status)
VALUES ($1,'TRANSACTION_STATUS_CHANGED',$2,$3::jsonb,$4,'PENDING') RETURNING id`,
		"routing-status-source-later-"+suffix, txKey, laterEnvelope, laterDigest,
	).Scan(&laterWebhookID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.RouteVerifiedEvent(ctx, VerifiedEventInput{
		WebhookEventID: laterWebhookID, EventType: "TRANSACTION_STATUS_CHANGED", PayloadDigest: laterDigest,
		NetworkFamily: "EVM", Snapshot: later,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO safeheron_transaction_routing_status_checks
  (safeheron_tx_key,first_seen_at,attempt_count,next_check_at,last_checked_at,last_check_outcome,
   last_observed_status,last_provider_event_id)
VALUES ($1,now()-interval '2 hours',4,now()+interval '1 hour',now(),'OBSERVED','SUBMITTED',$2)`,
		txKey, providerEventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE safeheron_transaction_routing_status_checks
SET first_seen_at=now()-interval '25 hours',last_observed_status='BROADCASTING'
WHERE safeheron_tx_key=$1`, txKey); err != nil {
		t.Fatal(err)
	}

	escalator, _ := NewAlertEscalator(db)
	worked, err := escalator.ProcessOne(ctx)
	if err != nil {
		t.Fatalf("ProcessOne() = %v, %v", worked, err)
	}
	if worked {
		t.Fatal("newly broadcast transaction inherited its approval-stage age")
	}
	if _, err := db.Exec(`UPDATE safeheron_transaction_routing_case_sources
SET linked_at=now()-interval '2 hours'
WHERE case_id=$1 AND upper(provider_status) IN ('BROADCASTING','CONFIRMING')`, caseID); err != nil {
		t.Fatal(err)
	}
	worked, err = escalator.ProcessOne(ctx)
	if err != nil || !worked {
		t.Fatalf("ProcessOne() after on-chain SLA = %v, %v", worked, err)
	}
	var subStatus, scope string
	var hasStartedAt bool
	var stuckSeconds int64
	if err := db.QueryRow(`SELECT payload->>'transaction_sub_status',payload->>'sla_scope',
       payload ? 'sla_started_at',(payload->>'stuck_seconds')::bigint
FROM safeheron_transaction_routing_alerts
WHERE case_id=$1 AND alert_type='SLA_ESCALATION' AND transition_key='sla:onchain:level:1'`, caseID).
		Scan(&subStatus, &scope, &hasStartedAt, &stuckSeconds); err != nil {
		t.Fatal(err)
	}
	if subStatus != "API_OBSERVED" {
		t.Fatalf("alert substatus=%q, want canonical API observation", subStatus)
	}
	if scope != "ONCHAIN" || !hasStartedAt || stuckSeconds < 2*60*60 {
		t.Fatalf("on-chain SLA payload scope=%q started=%v stuck=%d", scope, hasStartedAt, stuckSeconds)
	}
}

func TestAlertEscalatorPostgresKeepsPreChainApprovalStatusQuiet(t *testing.T) {
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
	t.Cleanup(func() { _ = db.Close() })
	cleanupRoutingAlertIntegrationFixtures(t, db)
	t.Cleanup(func() { cleanupRoutingAlertIntegrationFixtures(t, db) })

	ctx := context.Background()
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	txKey := "routing-alert-integration-prechain-" + suffix
	snapshot := routingSnapshot()
	snapshot.TxKey = txKey
	snapshot.TxHash = ""
	snapshot.TransactionDirection = "OUTFLOW"
	snapshot.TransactionStatus = "SUBMITTED"
	snapshot.TransactionSubStatus = "PENDING_APPROVAL"
	snapshot.CreateTime = time.Now().Add(-2 * time.Hour).UnixMilli()
	snapshotBody, _ := json.Marshal(snapshot)
	snapshot.RawPayload = snapshotBody
	envelope, _ := json.Marshal(map[string]any{
		"eventType": "TRANSACTION_CREATED", "eventDetail": snapshot,
	})
	eventID := "routing-alert-integration-prechain-event-" + suffix
	digest := fmt.Sprintf("%064x", time.Now().UnixNano())
	var webhookID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO safeheron_webhook_events
  (event_id,event_type,safeheron_tx_key,raw_payload,payload_digest,process_status)
VALUES ($1,'TRANSACTION_CREATED',$2,$3::jsonb,$4,'PENDING') RETURNING id`,
		eventID, txKey, envelope, digest,
	).Scan(&webhookID); err != nil {
		t.Fatal(err)
	}
	results, err := NewRepository(db).RouteVerifiedEvent(ctx, VerifiedEventInput{
		WebhookEventID: webhookID, EventType: "TRANSACTION_CREATED", PayloadDigest: digest,
		NetworkFamily: "EVM", Snapshot: snapshot,
	})
	if err != nil || len(results) != 1 || results[0].Decision.Reason != ReasonStatusNotTerminal {
		t.Fatalf("initial route = %#v, %v", results, err)
	}
	caseID := results[0].CaseID
	if _, err := db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_cases
SET created_at=now()-interval '2 hours' WHERE id=$1`, caseID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_status_checks
  (safeheron_tx_key,first_seen_at,attempt_count,next_check_at,last_checked_at,last_check_outcome,
   last_observed_status,last_provider_event_id)
VALUES ($1,now()-interval '2 hours',4,now()+interval '1 hour',now(),'OBSERVED','SUBMITTED',$2)`,
		txKey, eventID,
	); err != nil {
		t.Fatal(err)
	}

	escalator, err := NewAlertEscalator(db)
	if err != nil {
		t.Fatal(err)
	}
	worked, err := escalator.ProcessOne(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if worked {
		t.Fatal("SUBMITTED approval-stage transaction emitted an SLA alert")
	}
	var alertCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*)
FROM safeheron_transaction_routing_alerts WHERE case_id=$1`, caseID).Scan(&alertCount); err != nil {
		t.Fatal(err)
	}
	if alertCount != 0 {
		t.Fatalf("approval-stage alerts=%d, want 0", alertCount)
	}

	signing := snapshot
	signing.TransactionStatus = "SIGNING"
	signing.TransactionSubStatus = "WAITING_FOR_SIGNATURE"
	signingBody, _ := json.Marshal(signing)
	signing.RawPayload = signingBody
	signingEnvelope, _ := json.Marshal(map[string]any{
		"eventType": "TRANSACTION_STATUS_CHANGED", "eventDetail": signing,
	})
	signingEventID := "routing-alert-integration-signing-event-" + suffix
	signingDigest := fmt.Sprintf("%064x", time.Now().UnixNano()+1)
	var signingWebhookID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO safeheron_webhook_events
  (event_id,event_type,safeheron_tx_key,raw_payload,payload_digest,process_status)
VALUES ($1,'TRANSACTION_STATUS_CHANGED',$2,$3::jsonb,$4,'PENDING') RETURNING id`,
		signingEventID, txKey, signingEnvelope, signingDigest,
	).Scan(&signingWebhookID); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRepository(db).RouteVerifiedEvent(ctx, VerifiedEventInput{
		WebhookEventID: signingWebhookID, EventType: "TRANSACTION_STATUS_CHANGED",
		PayloadDigest: signingDigest, NetworkFamily: "EVM", Snapshot: signing,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_status_checks
SET last_checked_at=now(),last_check_outcome='OBSERVED',last_observed_status='SIGNING',
    last_provider_event_id=$2
WHERE safeheron_tx_key=$1`, txKey, signingEventID); err != nil {
		t.Fatal(err)
	}
	worked, err = escalator.ProcessOne(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if worked {
		t.Fatal("SIGNING transaction emitted an SLA alert")
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*)
FROM safeheron_transaction_routing_alerts WHERE case_id=$1`, caseID).Scan(&alertCount); err != nil {
		t.Fatal(err)
	}
	if alertCount != 0 {
		t.Fatalf("signing-stage alerts=%d, want 0", alertCount)
	}
}

func TestAlertEscalatorPostgresLateStatusCaseEmitsOnlyCurrentHighestLevel(t *testing.T) {
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
	t.Cleanup(func() { _ = db.Close() })
	cleanupRoutingAlertIntegrationFixtures(t, db)
	t.Cleanup(func() { cleanupRoutingAlertIntegrationFixtures(t, db) })

	ctx := context.Background()
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	txKey := "routing-alert-integration-catchup-" + suffix
	snapshot := routingSnapshot()
	snapshot.TxKey = txKey
	snapshot.TxHash = ""
	snapshot.TxHash = "0x" + suffix
	snapshot.TransactionStatus = "CONFIRMING"
	snapshot.TransactionSubStatus = "WAITING_FOR_CONFIRMATIONS"
	snapshot.CreateTime = time.Now().Add(-25 * time.Hour).UnixMilli()
	snapshotBody, _ := json.Marshal(snapshot)
	snapshot.RawPayload = snapshotBody
	envelope, _ := json.Marshal(map[string]any{
		"eventType": "TRANSACTION_CREATED", "eventDetail": snapshot,
	})
	eventID := "routing-alert-integration-catchup-event-" + suffix
	digest := fmt.Sprintf("%064x", time.Now().UnixNano())
	var webhookID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO safeheron_webhook_events
  (event_id,event_type,safeheron_tx_key,raw_payload,payload_digest,process_status)
VALUES ($1,'TRANSACTION_CREATED',$2,$3::jsonb,$4,'PENDING') RETURNING id`,
		eventID, txKey, envelope, digest,
	).Scan(&webhookID); err != nil {
		t.Fatal(err)
	}
	results, err := NewRepository(db).RouteVerifiedEvent(ctx, VerifiedEventInput{
		WebhookEventID: webhookID, EventType: "TRANSACTION_CREATED", PayloadDigest: digest,
		NetworkFamily: "EVM", Snapshot: snapshot,
	})
	if err != nil || len(results) != 1 || results[0].Decision.Reason != ReasonStatusNotTerminal {
		t.Fatalf("initial route = %#v, %v", results, err)
	}
	caseID := results[0].CaseID
	if _, err := db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_cases
SET created_at=now()-interval '25 hours' WHERE id=$1`, caseID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE safeheron_transaction_routing_case_sources
SET linked_at=now()-interval '25 hours' WHERE case_id=$1`, caseID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_status_checks
  (safeheron_tx_key,first_seen_at,attempt_count,next_check_at,last_checked_at,last_check_outcome,
   last_observed_status,last_provider_event_id)
VALUES ($1,now()-interval '25 hours',8,now()+interval '1 hour',now(),'OBSERVED','CONFIRMING',$2)`,
		txKey, eventID,
	); err != nil {
		t.Fatal(err)
	}

	escalator, err := NewAlertEscalator(db)
	if err != nil {
		t.Fatal(err)
	}
	worked, err := escalator.ProcessOne(ctx)
	if err != nil || !worked {
		t.Fatalf("first ProcessOne() = %v, %v", worked, err)
	}
	worked, err = escalator.ProcessOne(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if worked {
		t.Fatal("late catch-up emitted a lower SLA level after the current highest level")
	}

	var transitionKey, severity string
	var alertCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*),min(transition_key),min(severity)
FROM safeheron_transaction_routing_alerts
WHERE case_id=$1 AND alert_type='SLA_ESCALATION'`, caseID).
		Scan(&alertCount, &transitionKey, &severity); err != nil {
		t.Fatal(err)
	}
	if alertCount != 1 || transitionKey != "sla:onchain:level:3" || severity != "CRITICAL" {
		t.Fatalf("catch-up alerts=%d transition=%q severity=%q, want one level-3 CRITICAL", alertCount, transitionKey, severity)
	}
}
