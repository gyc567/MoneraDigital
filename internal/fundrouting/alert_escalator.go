package fundrouting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"monera-digital/internal/adaptiveschedule"
)

type AlertEscalator struct {
	db             *sql.DB
	runner         *adaptiveRunner
	onAlertCreated func()
	environment    string
}

// SetOnAlertCreated registers a wake after a durable SLA alert is inserted.
func (e *AlertEscalator) SetOnAlertCreated(fn func()) {
	if e == nil {
		return
	}
	e.onAlertCreated = fn
}

func (e *AlertEscalator) SetEnvironment(value string) {
	if e == nil {
		return
	}
	e.environment = strings.TrimSpace(value)
}

// Notify wakes SLA evaluation after a fresh provider-status observation.
func (e *AlertEscalator) Notify() bool {
	if e == nil || e.runner == nil {
		return false
	}
	return e.runner.Notify()
}

func NewAlertEscalator(db *sql.DB) (*AlertEscalator, error) {
	if db == nil {
		return nil, fmt.Errorf("fund routing alert escalation database is required")
	}
	e := &AlertEscalator{db: db, environment: "unknown"}
	// Pending-provider SLA thresholds are 1h/6h/24h; other OPEN reasons retain
	// 1h/24h. Min idle can be 1m while fully idle backs off to MaxIdle.
	e.runner = newAdaptiveRunner("fund routing OPEN-case SLA escalator", time.Minute, adaptiveschedule.DefaultMaxIdle, e.ProcessOne)
	e.runner.setNextDue(e.NextDue)
	return e, nil
}

// NextDue returns the earliest not-yet-emitted SLA threshold for an OPEN case.
func (e *AlertEscalator) NextDue(ctx context.Context) (time.Time, error) {
	var due sql.NullTime
	err := e.db.QueryRowContext(ctx, openCaseSLANextDueSQL()).Scan(&due)
	if err != nil || !due.Valid {
		return time.Time{}, err
	}
	return due.Time, nil
}

func openCaseSLANextDueSQL() string {
	return `WITH thresholds(reason_filter,level,minimum_age) AS (
  VALUES ('STATUS_NOT_TERMINAL'::varchar,1,interval '1 hour'),
         ('STATUS_NOT_TERMINAL'::varchar,2,interval '6 hours'),
         ('STATUS_NOT_TERMINAL'::varchar,3,interval '24 hours'),
         ('*'::varchar,1,interval '1 hour'),
         ('*'::varchar,2,interval '24 hours')
)
SELECT min((CASE WHEN threshold.reason_filter='STATUS_NOT_TERMINAL'
             THEN routing.created_at
             ELSE COALESCE(recovery.started_at,routing.created_at)
           END) + threshold.minimum_age)
FROM safeheron_transaction_routing_cases routing
CROSS JOIN thresholds threshold
LEFT JOIN safeheron_transaction_routing_status_checks status_check
  ON status_check.safeheron_tx_key=routing.safeheron_tx_key
LEFT JOIN LATERAL (
  SELECT max(alert.created_at) AS started_at
  FROM safeheron_transaction_routing_alerts alert
  WHERE alert.case_id=routing.id AND alert.alert_type='RECOVERY_SUMMARY'
    AND alert.transition_key LIKE 'sla:recovered:version:%'
) recovery ON true
WHERE routing.decision='OPEN'
  AND ((threshold.reason_filter='STATUS_NOT_TERMINAL' AND routing.reason_code=threshold.reason_filter)
    OR (threshold.reason_filter='*' AND routing.reason_code<>'STATUS_NOT_TERMINAL'))
  AND (CASE WHEN threshold.reason_filter='STATUS_NOT_TERMINAL'
         THEN routing.created_at
         ELSE COALESCE(recovery.started_at,routing.created_at)
       END) + threshold.minimum_age > now()
  AND (routing.reason_code<>'STATUS_NOT_TERMINAL'
    OR status_check.last_checked_at >= routing.created_at + threshold.minimum_age)
  AND (routing.reason_code<>'STATUS_NOT_TERMINAL'
    OR status_check.last_observed_status IS NULL
    OR status_check.last_observed_status NOT IN ('COMPLETED','FAILED','CANCELLED','REJECTED'))
  AND NOT EXISTS (
    SELECT 1 FROM safeheron_transaction_routing_alerts alert
    WHERE alert.case_id=routing.id AND alert.alert_type='SLA_ESCALATION'
      AND alert.transition_key=CASE WHEN threshold.reason_filter='STATUS_NOT_TERMINAL'
        THEN 'sla:pending:level:' ELSE 'sla:open:level:' END || threshold.level::text
  )`
}

func openCaseSLAEscalationSQL() string {
	return `WITH thresholds(reason_filter,level,minimum_age,severity) AS (
  VALUES ('STATUS_NOT_TERMINAL'::varchar,1,interval '1 hour','WARN'::varchar),
         ('STATUS_NOT_TERMINAL'::varchar,2,interval '6 hours','ERROR'::varchar),
         ('STATUS_NOT_TERMINAL'::varchar,3,interval '24 hours','CRITICAL'::varchar),
         ('*'::varchar,1,interval '1 hour','ERROR'::varchar),
         ('*'::varchar,2,interval '24 hours','CRITICAL'::varchar)
), candidate AS (
  SELECT routing.id AS case_id,routing.reason_code,threshold.reason_filter,
         threshold.level,threshold.severity,$1::text AS environment,
         routing.safeheron_tx_key,routing.raw_coin_key,routing.network_family,
         routing.direction,routing.movement_kind,routing.amount,
         routing.normalized_source AS source_address,
         routing.normalized_destination AS destination_address,
         routing.effective_event_time,
         floor(extract(epoch FROM now()-routing.created_at))::bigint AS stuck_seconds,
         source.provider_status,
         webhook.event_type AS last_source_event_type,
         webhook.received_at AS last_source_received_at,
         COALESCE(webhook.raw_payload#>>'{eventDetail,transactionSubStatus}',
                  webhook.raw_payload->>'transactionSubStatus','') AS transaction_sub_status,
         COALESCE(webhook.raw_payload#>>'{eventDetail,txHash}',
                  webhook.raw_payload->>'txHash','') AS tx_hash,
         status_check.last_checked_at AS last_api_checked_at,
         status_check.last_check_outcome AS last_api_check_outcome,
         status_check.last_error_code AS last_api_error_code
  FROM safeheron_transaction_routing_cases routing
  CROSS JOIN thresholds threshold
  LEFT JOIN safeheron_transaction_routing_status_checks status_check
    ON status_check.safeheron_tx_key=routing.safeheron_tx_key
  LEFT JOIN LATERAL (
    SELECT max(alert.created_at) AS started_at
    FROM safeheron_transaction_routing_alerts alert
    WHERE alert.case_id=routing.id AND alert.alert_type='RECOVERY_SUMMARY'
      AND alert.transition_key LIKE 'sla:recovered:version:%'
  ) recovery ON true
  JOIN LATERAL (
    SELECT linked.provider_status,linked.safeheron_webhook_event_id
    FROM safeheron_transaction_routing_case_sources linked
    JOIN safeheron_webhook_events linked_webhook
      ON linked_webhook.id=linked.safeheron_webhook_event_id
    WHERE linked.case_id=routing.id
	  AND (routing.reason_code<>'STATUS_NOT_TERMINAL'
	    OR status_check.last_check_outcome='ERROR'
	    OR linked_webhook.event_id=status_check.last_provider_event_id)
    ORDER BY linked.provider_status_rank DESC,linked.id DESC
    LIMIT 1
  ) source ON true
  JOIN safeheron_webhook_events webhook ON webhook.id=source.safeheron_webhook_event_id
  WHERE routing.decision='OPEN'
    AND (CASE WHEN threshold.reason_filter='STATUS_NOT_TERMINAL'
           THEN routing.created_at
           ELSE COALESCE(recovery.started_at,routing.created_at)
         END) <= now()-threshold.minimum_age
    AND ((threshold.reason_filter='STATUS_NOT_TERMINAL' AND routing.reason_code=threshold.reason_filter)
      OR (threshold.reason_filter='*' AND routing.reason_code<>'STATUS_NOT_TERMINAL'))
    AND (routing.reason_code<>'STATUS_NOT_TERMINAL'
      OR status_check.last_checked_at >= routing.created_at + threshold.minimum_age)
    AND (routing.reason_code<>'STATUS_NOT_TERMINAL'
      OR status_check.last_check_outcome='ERROR'
      OR (status_check.last_check_outcome='OBSERVED'
        AND webhook.event_id=status_check.last_provider_event_id))
    AND (routing.reason_code<>'STATUS_NOT_TERMINAL'
      OR status_check.last_observed_status IS NULL
      OR status_check.last_observed_status NOT IN ('COMPLETED','FAILED','CANCELLED','REJECTED'))
    AND NOT EXISTS (
      SELECT 1 FROM safeheron_transaction_routing_alerts alert
      WHERE alert.case_id=routing.id AND alert.alert_type='SLA_ESCALATION'
        AND alert.transition_key=CASE WHEN threshold.reason_filter='STATUS_NOT_TERMINAL'
          THEN 'sla:pending:level:' ELSE 'sla:open:level:' END || threshold.level::text
    )
    AND (threshold.reason_filter<>'STATUS_NOT_TERMINAL' OR NOT EXISTS (
      SELECT 1
      FROM safeheron_transaction_routing_alerts alert
      JOIN thresholds emitted
        ON emitted.reason_filter='STATUS_NOT_TERMINAL'
       AND emitted.level>threshold.level
       AND alert.transition_key='sla:pending:level:' || emitted.level::text
      WHERE alert.case_id=routing.id AND alert.alert_type='SLA_ESCALATION'
    ))
  ORDER BY routing.created_at, routing.id,
           CASE WHEN threshold.reason_filter='STATUS_NOT_TERMINAL' THEN threshold.level END DESC,
           threshold.level
  LIMIT 1
)
INSERT INTO safeheron_transaction_routing_alerts
  (case_id,alert_type,transition_key,severity,payload)
SELECT case_id,'SLA_ESCALATION',
       CASE WHEN reason_filter='STATUS_NOT_TERMINAL'
         THEN 'sla:pending:level:' ELSE 'sla:open:level:' END || level::text,
       severity,jsonb_strip_nulls(jsonb_build_object(
         'environment',environment,
         'case_id',case_id,
         'level',level,
         'reason_code',reason_code,
         'safeheron_tx_key',safeheron_tx_key,
         'raw_coin_key',raw_coin_key,
         'network_family',network_family,
         'direction',direction,
         'movement_kind',movement_kind,
         'amount',amount::text,
         'source_address',source_address,
         'destination_address',destination_address,
         'transaction_status',provider_status,
         'transaction_sub_status',transaction_sub_status,
         'tx_hash',tx_hash,
         'effective_event_time',effective_event_time,
         'stuck_seconds',stuck_seconds,
         'last_source_event_type',last_source_event_type,
         'last_source_received_at',last_source_received_at,
         'last_api_checked_at',last_api_checked_at,
         'last_api_check_outcome',last_api_check_outcome,
         'last_api_error_code',last_api_error_code
       ))
FROM candidate
ON CONFLICT (case_id,alert_type,transition_key) DO NOTHING
RETURNING id`
}

func (e *AlertEscalator) ProcessOne(ctx context.Context) (bool, error) {
	var alertID int64
	environment := e.environment
	if environment == "" {
		environment = "unknown"
	}
	err := e.db.QueryRowContext(ctx, openCaseSLAEscalationSQL(), environment).Scan(&alertID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if e.onAlertCreated != nil {
		e.onAlertCreated()
	}
	return true, nil
}

func (e *AlertEscalator) Run(ctx context.Context) {
	if e == nil || e.runner == nil {
		return
	}
	e.runner.Run(ctx)
}
