package fundrouting

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAlertEscalatorNextDueReadsEarliestMissingSLAThreshold(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	escalator, err := NewAlertEscalator(db)
	if err != nil {
		t.Fatal(err)
	}
	due := time.Now().Add(time.Hour).Round(time.Microsecond)
	mock.ExpectQuery("SELECT min\\(\\(CASE WHEN threshold.reason_filter").
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(due))

	got, err := escalator.NextDue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(due) {
		t.Fatalf("NextDue=%s, want %s", got, due)
	}
}

func TestAlertEscalatorCreatesAtMostOneMissingOpenSLALevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	escalator, err := NewAlertEscalator(db)
	if err != nil {
		t.Fatal(err)
	}
	wakeCount := 0
	escalator.SetOnAlertCreated(func() { wakeCount++ })
	mock.ExpectQuery("INSERT INTO safeheron_transaction_routing_alerts").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(8))
	processed, err := escalator.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOne = %v, %v", processed, err)
	}
	if wakeCount != 1 {
		t.Fatalf("notifier wakes=%d, want 1 after alert insert", wakeCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertEscalatorReturnsIdleWhenNoThresholdIsDue(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	escalator, _ := NewAlertEscalator(db)
	wakeCount := 0
	escalator.SetOnAlertCreated(func() { wakeCount++ })
	mock.ExpectQuery("INSERT INTO safeheron_transaction_routing_alerts").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	processed, err := escalator.ProcessOne(context.Background())
	if err != nil || processed {
		t.Fatalf("ProcessOne = %v, %v", processed, err)
	}
	if wakeCount != 0 {
		t.Fatalf("idle escalator emitted %d notifier wakes", wakeCount)
	}
}

func TestAlertEscalatorSLAThresholdsDifferentiateOnChainProviderStatus(t *testing.T) {
	sqlText := openCaseSLAEscalationSQL()
	for _, fragment := range []string{
		"('STATUS_NOT_TERMINAL'::varchar,1,interval '1 hour','WARN'::varchar)",
		"('STATUS_NOT_TERMINAL'::varchar,2,interval '6 hours','ERROR'::varchar)",
		"('STATUS_NOT_TERMINAL'::varchar,3,interval '24 hours','CRITICAL'::varchar)",
		"('*'::varchar,1,interval '1 hour','ERROR'::varchar)",
		"('*'::varchar,2,interval '24 hours','CRITICAL'::varchar)",
	} {
		if !strings.Contains(sqlText, fragment) {
			t.Errorf("SLA escalation SQL is missing threshold %q", fragment)
		}
	}
}

func TestAlertEscalatorStartsOpenReasonSLAFromProviderRecovery(t *testing.T) {
	for name, sqlText := range map[string]string{
		"next due":   openCaseSLANextDueSQL(),
		"escalation": openCaseSLAEscalationSQL(),
	} {
		for _, fragment := range []string{
			"alert.alert_type='RECOVERY_SUMMARY'",
			"alert.transition_key LIKE 'sla:recovered:version:%'",
			"COALESCE(recovery.started_at,routing.created_at)",
		} {
			if !strings.Contains(sqlText, fragment) {
				t.Errorf("%s SQL is missing recovery SLA anchor %q", name, fragment)
			}
		}
	}
}

func TestAlertEscalatorRequiresFreshStatusCheckAndBuildsActionablePayload(t *testing.T) {
	sqlText := openCaseSLAEscalationSQL()
	for _, fragment := range []string{
		"status_check.last_checked_at >= chain_progress.started_at + threshold.minimum_age",
		"status_check.last_check_outcome='ERROR'",
		"linked_webhook.event_id=status_check.last_provider_event_id",
		"status_check.last_observed_status IN ('BROADCASTING','CONFIRMING')",
		"upper(linked.provider_status) IN ('BROADCASTING','CONFIRMING')",
		"webhook.received_at AT TIME ZONE 'UTC' AS last_source_received_at",
		"'sla:onchain:level:'",
		"'environment',environment",
		"'case_id',case_id",
		"'safeheron_tx_key',safeheron_tx_key",
		"'raw_coin_key',raw_coin_key",
		"'network_family',network_family",
		"'amount',amount::text",
		"'source_address',source_address",
		"'destination_address',destination_address",
		"'transaction_status',provider_status",
		"'transaction_sub_status',transaction_sub_status",
		"'tx_hash',tx_hash",
		"'effective_event_time',effective_event_time",
		"'sla_started_at',sla_started_at",
		"'stuck_seconds',stuck_seconds",
		"'last_source_event_type',last_source_event_type",
		"'last_source_received_at',last_source_received_at",
		"'last_api_checked_at',last_api_checked_at",
		"'last_api_check_outcome',last_api_check_outcome",
		"'last_api_error_code',last_api_error_code",
	} {
		if !strings.Contains(sqlText, fragment) {
			t.Errorf("SLA escalation SQL is missing actionable field/gate %q", fragment)
		}
	}
	if strings.Contains(sqlText, "status_check.safeheron_tx_key IS NULL") {
		t.Fatal("STATUS_NOT_TERMINAL escalation must not bypass the provider lookup when no check exists")
	}
	if strings.Contains(openCaseSLANextDueSQL(), "status_check.safeheron_tx_key IS NULL") {
		t.Fatal("STATUS_NOT_TERMINAL scheduling must wait for a durable provider check")
	}
	if strings.Contains(sqlText, "sla:pending:level:") || strings.Contains(openCaseSLANextDueSQL(), "sla:pending:level:") {
		t.Fatal("approval-stage SLA transition keys must not remain active")
	}
}
