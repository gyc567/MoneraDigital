package fundrouting

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"monera-digital/internal/alert"
)

func TestAlertNotifierNextDueReadsEarliestDurableRetryOrLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	notifier, err := NewAlertNotifier(db, routingAlertSenderStub{})
	if err != nil {
		t.Fatal(err)
	}
	due := time.Now().Add(30 * time.Second).Round(time.Microsecond)
	mock.ExpectQuery("SELECT min\\(due_at\\)").WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(due))

	got, err := notifier.NextDue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(due) {
		t.Fatalf("NextDue=%s, want %s", got, due)
	}
}

func TestRoutingAlertPresentationExplainsPendingTransactionsWithCompleteIdentifiers(t *testing.T) {
	payload := map[string]any{
		"environment": "production", "case_id": 879, "reason_code": "STATUS_NOT_TERMINAL",
		"safeheron_tx_key": "tx-key-complete", "raw_coin_key": "USDT_ERC20", "network_family": "EVM",
		"direction": "OUTFLOW", "movement_kind": "PRINCIPAL", "amount": "9629.63",
		"source_address":      "0x1111111111111111111111111111111111111111",
		"destination_address": "0x2222222222222222222222222222222222222222",
		"transaction_status":  "SUBMITTED", "transaction_sub_status": "PENDING_APPROVAL",
		"tx_hash": "", "effective_event_time": "2026-08-10T03:26:03Z", "stuck_seconds": 3600,
		"last_source_event_type": "TRANSACTION_CREATED", "last_source_received_at": "2026-08-10T03:26:04Z",
		"last_api_checked_at": "2026-08-10T04:26:03Z", "last_api_check_outcome": "OBSERVED",
	}
	raw, _ := json.Marshal(payload)
	delivery := claimedDelivery{Severity: "WARN", AlertType: "SLA_ESCALATION", Payload: raw}

	severity, title, fields := routingAlertPresentation([]claimedDelivery{delivery})
	if severity != "WARN" || title != "Safeheron 出账停留超时（1笔）" {
		t.Fatalf("presentation header = %s / %s", severity, title)
	}
	if fields["环境"] != "production" || fields["交易数量"] != "1" || !strings.Contains(fields["告警原因"], "尚未进入终态") {
		t.Fatalf("summary fields = %#v", fields)
	}
	detail := fields["交易01"]
	for _, fragment := range []string{
		"Case ID：879", "方向：出账", "资产：USDT_ERC20", "网络：EVM", "金额：9629.63",
		"状态：SUBMITTED（已提交，尚未进入终态）", "子状态：PENDING_APPROVAL", "TxKey：tx-key-complete",
		"Tx Hash：尚未广播", "来源地址：0x1111111111111111111111111111111111111111",
		"目标地址：0x2222222222222222222222222222222222222222", "停留时长：1小时",
		"最后事件：TRANSACTION_CREATED", "最后 API 核验：2026-08-10 12:26:03 UTC+8",
	} {
		if !strings.Contains(detail, fragment) {
			t.Errorf("transaction detail is missing %q:\n%s", fragment, detail)
		}
	}
}

func TestRoutingAlertPresentationAggregatesFiveMinuteLarkBatch(t *testing.T) {
	first, _ := json.Marshal(map[string]any{
		"case_id": 1, "reason_code": "STATUS_NOT_TERMINAL", "direction": "OUTFLOW",
		"safeheron_tx_key": "tx-1", "transaction_status": "SUBMITTED",
	})
	second, _ := json.Marshal(map[string]any{
		"case_id": 2, "reason_code": "STATUS_NOT_TERMINAL", "direction": "OUTFLOW",
		"safeheron_tx_key": "tx-2", "transaction_status": "SIGNING",
	})
	severity, title, fields := routingAlertPresentation([]claimedDelivery{
		{Severity: "WARN", AlertType: "SLA_ESCALATION", Payload: first},
		{Severity: "ERROR", AlertType: "SLA_ESCALATION", Payload: second},
	})
	if severity != "ERROR" || title != "Safeheron 出账停留超时（2笔）" || fields["交易数量"] != "2" || fields["交易01"] == "" || fields["交易02"] == "" {
		t.Fatalf("aggregate presentation = %s / %s / %#v", severity, title, fields)
	}
}

func TestRoutingAlertPresentationExplainsRecoveredTransaction(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"environment": "production", "case_id": 879, "direction": "OUTFLOW",
		"safeheron_tx_key": "tx-key-complete", "raw_coin_key": "USDT_ERC20",
		"network_family": "EVM", "amount": "9629.63",
		"source_address":      "0x1111111111111111111111111111111111111111",
		"destination_address": "0x2222222222222222222222222222222222222222",
		"transaction_status":  "COMPLETED", "tx_hash": "0xfull-hash",
		"resolved_decision": "COMPANY", "resolved_reason_code": "COMPANY_MATCHED",
	})

	severity, title, fields := routingAlertPresentation([]claimedDelivery{{
		Severity: "INFO", AlertType: "RECOVERY_SUMMARY", Payload: raw,
	}})
	if severity != "INFO" || title != "Safeheron 出账状态已收敛（1笔）" {
		t.Fatalf("recovery presentation header = %s / %s", severity, title)
	}
	if fields["恢复说明"] == "" || fields["交易数量"] != "1" {
		t.Fatalf("recovery summary fields = %#v", fields)
	}
	detail := fields["交易01"]
	for _, fragment := range []string{
		"状态：COMPLETED（已完成）", "TxKey：tx-key-complete", "Tx Hash：0xfull-hash",
		"处理结果：COMPANY", "处理原因：COMPANY_MATCHED",
	} {
		if !strings.Contains(detail, fragment) {
			t.Errorf("recovery detail is missing %q:\n%s", fragment, detail)
		}
	}
}

type routingAlertSenderStub struct {
	sinks   []alert.RoutingSink
	outcome alert.RoutingDeliveryOutcome
}

type routingAlertSenderRecorder struct {
	calls    int
	kind     string
	severity string
	title    string
	fields   map[string]string
}

func (s *routingAlertSenderRecorder) RoutingSinks() []alert.RoutingSink {
	return []alert.RoutingSink{{Kind: "LARK", Fingerprint: strings.Repeat("a", 64)}}
}

func (s *routingAlertSenderRecorder) SendRouting(_ context.Context, kind, _ string, severity, title string, fields map[string]string) alert.RoutingDeliveryOutcome {
	s.calls++
	s.kind = kind
	s.severity = severity
	s.title = title
	s.fields = fields
	return alert.RoutingDeliverySent
}

type routingAlertQueueStub struct {
	batch    []claimedDelivery
	finished []claimedDelivery
	outcome  alert.RoutingDeliveryOutcome
}

func (s *routingAlertQueueStub) NextDue(context.Context) (time.Time, error) {
	return time.Time{}, nil
}

func (s *routingAlertQueueStub) SweepExpired(context.Context) error { return nil }

func (s *routingAlertQueueStub) EnsureDeliveries(context.Context, []alert.RoutingSink, time.Duration) (bool, error) {
	return false, nil
}

func (s *routingAlertQueueStub) Claim(context.Context, string, int) ([]claimedDelivery, error) {
	batch := s.batch
	s.batch = nil
	return batch, nil
}

func (s *routingAlertQueueStub) Finish(_ context.Context, _ string, batch []claimedDelivery, outcome alert.RoutingDeliveryOutcome) error {
	s.finished = append(s.finished, batch...)
	s.outcome = outcome
	return nil
}

func TestLarkRoutingBatchDueUsesFixedFiveMinuteWindow(t *testing.T) {
	created := time.Date(2026, 8, 10, 4, 26, 3, 0, time.UTC)
	want := time.Date(2026, 8, 10, 4, 30, 0, 0, time.UTC)
	if got := larkRoutingBatchDue(created, 5*time.Minute); !got.Equal(want) {
		t.Fatalf("larkRoutingBatchDue() = %s, want %s", got, want)
	}
}

func TestAlertNotifierSendsOneLarkMessageForClaimedBatch(t *testing.T) {
	first, _ := json.Marshal(map[string]any{
		"case_id": 1, "reason_code": "STATUS_NOT_TERMINAL", "direction": "OUTFLOW",
		"safeheron_tx_key": "tx-1", "transaction_status": "SUBMITTED",
	})
	second, _ := json.Marshal(map[string]any{
		"case_id": 2, "reason_code": "STATUS_NOT_TERMINAL", "direction": "OUTFLOW",
		"safeheron_tx_key": "tx-2", "transaction_status": "SUBMITTED",
	})
	queue := &routingAlertQueueStub{batch: []claimedDelivery{
		{ID: 1, SinkKind: "LARK", Fingerprint: strings.Repeat("a", 64), Severity: "WARN", AlertType: "SLA_ESCALATION", Payload: first},
		{ID: 2, SinkKind: "LARK", Fingerprint: strings.Repeat("a", 64), Severity: "WARN", AlertType: "SLA_ESCALATION", Payload: second},
	}}
	sender := &routingAlertSenderRecorder{}
	notifier, err := newAlertNotifierWithQueue(queue, sender)
	if err != nil {
		t.Fatal(err)
	}
	worked, err := notifier.ProcessOne(context.Background())
	if err != nil || !worked {
		t.Fatalf("ProcessOne() = %v, %v", worked, err)
	}
	if sender.calls != 1 || sender.kind != "LARK" || sender.severity != "WARN" || sender.title != "Safeheron 出账停留超时（2笔）" || sender.fields["交易数量"] != "2" {
		t.Fatalf("sender call = %#v", sender)
	}
	if len(queue.finished) != 2 || queue.outcome != alert.RoutingDeliverySent {
		t.Fatalf("finished = %#v / %s", queue.finished, queue.outcome)
	}
}

func TestLarkBatchCombinesSeverityLevelsWithinTheSameAlertWindow(t *testing.T) {
	sqlText := claimableLarkBatchQuery()
	if !strings.Contains(sqlText, "alert.alert_type=$3") {
		t.Fatal("Lark batching must keep unrelated alert types separate")
	}
	if strings.Contains(sqlText, "alert.severity=") {
		t.Fatal("Lark batching must not split one five-minute window by severity")
	}
	if !strings.Contains(sqlText, "$6='FAILED_DEFINITE'") {
		t.Fatal("Lark batching must preserve a batch across automatic retries")
	}
	if !strings.Contains(sqlText, "LIMIT $7") {
		t.Fatal("Lark aggregation must use a bounded, stable chunk size")
	}
	if !strings.Contains(sqlText, "octet_length(alert.payload::text)") {
		t.Fatal("Lark aggregation must enforce a serialized payload budget")
	}
}

func (s routingAlertSenderStub) RoutingSinks() []alert.RoutingSink { return s.sinks }
func (s routingAlertSenderStub) SendRouting(context.Context, string, string, string, string, map[string]string) alert.RoutingDeliveryOutcome {
	return s.outcome
}

func TestAlertNotifierUnknownDeliveryBecomesAmbiguousWithoutRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	notifier, err := NewAlertNotifier(db, routingAlertSenderStub{})
	if err != nil {
		t.Fatal(err)
	}
	delivery := claimedDelivery{ID: 7, AttemptID: 8, Attempt: 1}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE safeheron_transaction_routing_alert_delivery_attempts").
		WithArgs(int64(8), "DELIVERY_UNKNOWN").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE safeheron_transaction_routing_alert_deliveries").
		WithArgs(int64(7), "AMBIGUOUS", true, false, notifier.workerID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := notifier.queue.Finish(context.Background(), notifier.workerID, []claimedDelivery{delivery}, alert.RoutingDeliveryUnknown); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertNotifierSweepsExpiredDispatchToAmbiguous(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	notifier, _ := NewAlertNotifier(db, routingAlertSenderStub{})
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE safeheron_transaction_routing_alert_delivery_attempts attempt").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE safeheron_transaction_routing_alert_deliveries").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := notifier.queue.SweepExpired(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertNotifierMaterializesAllCurrentSinksOnlyForAnUnfannedAlert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	notifier, _ := NewAlertNotifier(db, routingAlertSenderStub{sinks: []alert.RoutingSink{
		{Kind: "LARK", Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{Kind: "EMAIL", Fingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	}})
	createdAt := time.Date(2026, 8, 10, 4, 26, 3, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT alert.id").WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(5, createdAt))
	mock.ExpectExec("INSERT INTO safeheron_transaction_routing_alert_deliveries").
		WithArgs(int64(5), "LARK", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", larkRoutingBatchDue(createdAt, routingLarkBatchWindow)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO safeheron_transaction_routing_alert_deliveries").
		WithArgs(int64(5), "EMAIL", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", nil).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()
	if _, err := notifier.queue.EnsureDeliveries(context.Background(), notifier.sender.RoutingSinks(), routingLarkBatchWindow); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
