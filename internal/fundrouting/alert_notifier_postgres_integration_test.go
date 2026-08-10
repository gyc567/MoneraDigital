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

	"monera-digital/internal/alert"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type sequenceRoutingAlertSender struct {
	fingerprint string
	outcomes    []alert.RoutingDeliveryOutcome
	batchSizes  []string
}

func (s *sequenceRoutingAlertSender) RoutingSinks() []alert.RoutingSink {
	fingerprint := s.fingerprint
	if fingerprint == "" {
		fingerprint = strings.Repeat("e", 64)
	}
	return []alert.RoutingSink{{Kind: "LARK", Fingerprint: fingerprint}}
}

func (s *sequenceRoutingAlertSender) SendRouting(_ context.Context, _, _ string, _, _ string, fields map[string]string) alert.RoutingDeliveryOutcome {
	s.batchSizes = append(s.batchSizes, fields["交易数量"])
	outcome := s.outcomes[0]
	s.outcomes = s.outcomes[1:]
	return outcome
}

func cleanupRoutingAlertIntegrationFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rollback := func() { _ = tx.Rollback() }
	statements := []string{
		`DELETE FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id IN (
  SELECT delivery.id FROM safeheron_transaction_routing_alert_deliveries delivery
  JOIN safeheron_transaction_routing_alerts alert ON alert.id=delivery.alert_id
  JOIN safeheron_transaction_routing_cases routing ON routing.id=alert.case_id
  WHERE routing.safeheron_tx_key LIKE 'routing-alert-integration-%'
     OR routing.safeheron_tx_key LIKE 'routing-alert-batch-%')`,
		`DELETE FROM safeheron_transaction_routing_alert_deliveries WHERE alert_id IN (
  SELECT alert.id FROM safeheron_transaction_routing_alerts alert
  JOIN safeheron_transaction_routing_cases routing ON routing.id=alert.case_id
  WHERE routing.safeheron_tx_key LIKE 'routing-alert-integration-%'
     OR routing.safeheron_tx_key LIKE 'routing-alert-batch-%')`,
		`DELETE FROM safeheron_transaction_routing_alerts WHERE case_id IN (
  SELECT id FROM safeheron_transaction_routing_cases
  WHERE safeheron_tx_key LIKE 'routing-alert-integration-%'
     OR safeheron_tx_key LIKE 'routing-alert-batch-%')`,
		`DELETE FROM safeheron_transaction_routing_case_sources WHERE case_id IN (
  SELECT id FROM safeheron_transaction_routing_cases
  WHERE safeheron_tx_key LIKE 'routing-alert-integration-%'
     OR safeheron_tx_key LIKE 'routing-alert-batch-%')`,
		`DELETE FROM safeheron_transaction_routing_status_checks
  WHERE safeheron_tx_key LIKE 'routing-alert-integration-%'
     OR safeheron_tx_key LIKE 'routing-alert-batch-%'`,
		`DELETE FROM safeheron_transaction_routing_cases
  WHERE safeheron_tx_key LIKE 'routing-alert-integration-%'
     OR safeheron_tx_key LIKE 'routing-alert-batch-%'`,
		`DELETE FROM company_fund_safeheron_raw_event_exclusions WHERE safeheron_webhook_event_id IN (
  SELECT id FROM safeheron_webhook_events
  WHERE safeheron_tx_key LIKE 'routing-alert-integration-%'
     OR safeheron_tx_key LIKE 'routing-alert-batch-%')`,
		`DELETE FROM safeheron_webhook_events
  WHERE safeheron_tx_key LIKE 'routing-alert-integration-%'
     OR safeheron_tx_key LIKE 'routing-alert-batch-%'`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(context.Background(), statement); err != nil {
			rollback()
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertNotifierPostgresPreservesUnknownAttemptAcrossManualReplay(t *testing.T) {
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

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	snapshot := routingSnapshot()
	snapshot.TxKey = "routing-alert-integration-" + suffix
	snapshot.TxHash = "0x" + suffix
	snapshot.SourceAddress = "0x00000000000000000000000000000000000000a1"
	snapshot.DestinationAddress = "0x00000000000000000000000000000000000000b2"
	payload, _ := json.Marshal(map[string]any{"eventType": "TRANSACTION_STATUS_CHANGED", "eventDetail": snapshot})
	digest := strings.Repeat("d", 64)
	var webhookID, caseID, alertID, deliveryID int64
	if err := db.QueryRow(`INSERT INTO safeheron_webhook_events
  (event_id,event_type,safeheron_tx_key,raw_payload,payload_digest,process_status)
VALUES ($1,'TRANSACTION_STATUS_CHANGED',$2,$3::jsonb,$4,'PENDING') RETURNING id`, "routing-alert-event-"+suffix, snapshot.TxKey, payload, digest).Scan(&webhookID); err != nil {
		t.Fatal(err)
	}
	results, err := NewRepository(db).RouteVerifiedEvent(context.Background(), VerifiedEventInput{
		WebhookEventID: webhookID, EventType: "TRANSACTION_STATUS_CHANGED", PayloadDigest: digest,
		NetworkFamily: "EVM", Snapshot: snapshot,
	})
	if err != nil {
		t.Fatal(err)
	}
	caseID = results[0].CaseID
	if err := db.QueryRow(`SELECT id FROM safeheron_transaction_routing_alerts WHERE case_id=$1`, caseID).Scan(&alertID); err != nil {
		t.Fatal(err)
	}
	deliveryID = -time.Now().UnixNano()
	if _, err := db.Exec(`INSERT INTO safeheron_transaction_routing_alert_deliveries
	  (id,alert_id,sink_kind,recipient_fingerprint,created_at,updated_at)
VALUES ($1,$2,'LARK',$3,'2000-01-01T00:00:00Z','2000-01-01T00:00:00Z')`,
		deliveryID, alertID, strings.Repeat("e", 64)); err != nil {
		t.Fatal(err)
	}

	sender := &sequenceRoutingAlertSender{outcomes: []alert.RoutingDeliveryOutcome{
		alert.RoutingDeliveryUnknown,
		alert.RoutingDeliveryDefinitelyNotSent,
		alert.RoutingDeliverySent,
	}}
	notifier, _ := NewAlertNotifier(db, sender)
	processed, err := notifier.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("first process processed=%v err=%v", processed, err)
	}
	if err := db.QueryRow(`SELECT id FROM safeheron_transaction_routing_alert_deliveries WHERE alert_id=$1`, alertID).Scan(&deliveryID); err != nil {
		t.Fatal(err)
	}
	var status string
	var automaticAttempts int
	if err := db.QueryRow(`SELECT status,automatic_attempt_count FROM safeheron_transaction_routing_alert_deliveries WHERE id=$1`, deliveryID).Scan(&status, &automaticAttempts); err != nil {
		t.Fatal(err)
	}
	if status != "AMBIGUOUS" || automaticAttempts != 1 {
		t.Fatalf("after unknown status=%s automatic_attempts=%d", status, automaticAttempts)
	}

	if _, err := db.Exec(`INSERT INTO safeheron_transaction_routing_alert_delivery_attempts
  (delivery_id,attempt_number,attempt_kind,outcome,actor_admin_user_id,reason,idempotency_key)
VALUES ($1,2,'MANUAL_REPLAY','IN_PROGRESS',1,'integration replay','integration-replay')`, deliveryID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE safeheron_transaction_routing_alert_deliveries SET status='PENDING',completed_at=NULL WHERE id=$1`, deliveryID); err != nil {
		t.Fatal(err)
	}
	processed, err = notifier.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("manual replay processed=%v err=%v", processed, err)
	}
	if _, err := db.Exec(`UPDATE safeheron_transaction_routing_alert_deliveries
SET next_attempt_at=now() WHERE id=$1 AND status='FAILED_DEFINITE'`, deliveryID); err != nil {
		t.Fatal(err)
	}
	processed, err = notifier.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("automatic retry after manual failure processed=%v err=%v", processed, err)
	}

	var unknownAttempts, failedManualAttempts, sentAutoAttempts, totalAttempts, maxAttempt int
	if err := db.QueryRow(`SELECT status,automatic_attempt_count FROM safeheron_transaction_routing_alert_deliveries WHERE id=$1`, deliveryID).Scan(&status, &automaticAttempts); err != nil {
		t.Fatal(err)
	}
	_ = db.QueryRow(`SELECT count(*) FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id=$1`, deliveryID).Scan(&totalAttempts)
	_ = db.QueryRow(`SELECT count(*) FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id=$1 AND attempt_kind='AUTO' AND outcome='DELIVERY_UNKNOWN'`, deliveryID).Scan(&unknownAttempts)
	_ = db.QueryRow(`SELECT count(*) FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id=$1 AND attempt_kind='MANUAL_REPLAY' AND outcome='DEFINITELY_NOT_SENT'`, deliveryID).Scan(&failedManualAttempts)
	_ = db.QueryRow(`SELECT count(*) FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id=$1 AND attempt_kind='AUTO' AND outcome='SENT'`, deliveryID).Scan(&sentAutoAttempts)
	_ = db.QueryRow(`SELECT max(attempt_number) FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id=$1`, deliveryID).Scan(&maxAttempt)
	if status != "SENT" || automaticAttempts != 2 || totalAttempts != 3 || unknownAttempts != 1 || failedManualAttempts != 1 || sentAutoAttempts != 1 || maxAttempt != 3 {
		t.Fatalf("status=%s automatic=%d total=%d unknown=%d manual_failed=%d auto_sent=%d max_attempt=%d", status, automaticAttempts, totalAttempts, unknownAttempts, failedManualAttempts, sentAutoAttempts, maxAttempt)
	}

	if _, err := db.Exec(`UPDATE safeheron_transaction_routing_cases SET created_at=now()-interval '25 hours' WHERE id=$1`, caseID); err != nil {
		t.Fatal(err)
	}
	escalator, _ := NewAlertEscalator(db)
	for level := 1; level <= 2; level++ {
		processed, err = escalator.ProcessOne(context.Background())
		if err != nil || !processed {
			t.Fatalf("SLA level %d processed=%v err=%v", level, processed, err)
		}
	}
	var slaAlerts int
	if err := db.QueryRow(`SELECT count(*) FROM safeheron_transaction_routing_alerts WHERE case_id=$1 AND alert_type='SLA_ESCALATION'`, caseID).Scan(&slaAlerts); err != nil || slaAlerts != 2 {
		t.Fatalf("SLA alerts=%d err=%v", slaAlerts, err)
	}
}

func TestAlertNotifierPostgresBatchesMixedSeverityLarkAlertsInOneWindow(t *testing.T) {
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
	fingerprint := strings.Repeat("f", 64)
	due := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	var caseIDs, webhookIDs, deliveryIDs []int64

	for index, severity := range []string{"WARN", "ERROR"} {
		snapshot := routingSnapshot()
		snapshot.TxKey = fmt.Sprintf("routing-alert-batch-%s-%d", suffix, index)
		snapshot.TxHash = fmt.Sprintf("0xbatch%s%d", suffix, index)
		snapshot.SourceAddress = fmt.Sprintf("0x%040x", time.Now().UnixNano()+int64(index*2))
		snapshot.DestinationAddress = fmt.Sprintf("0x%040x", time.Now().UnixNano()+int64(index*2+1))
		payload, _ := json.Marshal(map[string]any{
			"eventType": "TRANSACTION_STATUS_CHANGED", "eventDetail": snapshot,
		})
		digest := fmt.Sprintf("%064x", time.Now().UnixNano()+int64(index))
		var webhookID int64
		if err := db.QueryRow(`INSERT INTO safeheron_webhook_events
  (event_id,event_type,safeheron_tx_key,raw_payload,payload_digest,process_status)
VALUES ($1,'TRANSACTION_STATUS_CHANGED',$2,$3::jsonb,$4,'PENDING') RETURNING id`,
			fmt.Sprintf("routing-alert-batch-event-%s-%d", suffix, index), snapshot.TxKey, payload, digest,
		).Scan(&webhookID); err != nil {
			t.Fatal(err)
		}
		webhookIDs = append(webhookIDs, webhookID)
		results, routeErr := NewRepository(db).RouteVerifiedEvent(ctx, VerifiedEventInput{
			WebhookEventID: webhookID, EventType: "TRANSACTION_STATUS_CHANGED",
			PayloadDigest: digest, NetworkFamily: "EVM", Snapshot: snapshot,
		})
		if routeErr != nil || len(results) != 1 {
			t.Fatalf("route result = %#v, %v", results, routeErr)
		}
		caseID := results[0].CaseID
		caseIDs = append(caseIDs, caseID)
		alertPayload, _ := json.Marshal(map[string]any{
			"case_id": caseID, "direction": "OUTFLOW", "safeheron_tx_key": snapshot.TxKey,
			"transaction_status": "SUBMITTED",
		})
		var alertID int64
		if err := db.QueryRow(`UPDATE safeheron_transaction_routing_alerts
SET alert_type='SLA_ESCALATION',transition_key=$2,severity=$3,payload=$4::jsonb
WHERE case_id=$1 RETURNING id`, caseID, fmt.Sprintf("batch:%d", index), severity, alertPayload).Scan(&alertID); err != nil {
			t.Fatal(err)
		}
		var deliveryID int64
		if err := db.QueryRow(`INSERT INTO safeheron_transaction_routing_alert_deliveries
  (alert_id,sink_kind,recipient_fingerprint,next_attempt_at)
VALUES ($1,'LARK',$2,$3) RETURNING id`, alertID, fingerprint, due).Scan(&deliveryID); err != nil {
			t.Fatal(err)
		}
		deliveryIDs = append(deliveryIDs, deliveryID)
	}

	sender := &sequenceRoutingAlertSender{fingerprint: fingerprint, outcomes: []alert.RoutingDeliveryOutcome{
		alert.RoutingDeliveryDefinitelyNotSent,
		alert.RoutingDeliverySent,
	}}
	notifier, _ := NewAlertNotifier(db, sender)
	worked, err := notifier.ProcessOne(ctx)
	if err != nil || !worked {
		t.Fatalf("ProcessOne() = %v, %v", worked, err)
	}
	if len(sender.batchSizes) != 1 || sender.batchSizes[0] != "2" {
		t.Fatalf("first sender batch sizes = %#v", sender.batchSizes)
	}
	if _, err := db.Exec(`UPDATE safeheron_transaction_routing_alert_deliveries
SET next_attempt_at=now() WHERE id=ANY($1) AND status='FAILED_DEFINITE'`, deliveryIDs); err != nil {
		t.Fatal(err)
	}
	worked, err = notifier.ProcessOne(ctx)
	if err != nil || !worked {
		t.Fatalf("retry ProcessOne() = %v, %v", worked, err)
	}
	if len(sender.batchSizes) != 2 || sender.batchSizes[1] != "2" {
		t.Fatalf("retry split the original Lark batch: %#v", sender.batchSizes)
	}
	var sent int
	if err := db.QueryRow(`SELECT count(*) FROM safeheron_transaction_routing_alert_deliveries
WHERE id=ANY($1) AND status='SENT'`, deliveryIDs).Scan(&sent); err != nil {
		t.Fatal(err)
	}
	if sent != 2 {
		t.Fatalf("sent deliveries=%d, want 2", sent)
	}
}
