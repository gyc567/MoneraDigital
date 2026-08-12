package fundrouting

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"monera-digital/internal/adaptiveschedule"
	"monera-digital/internal/safeheron"
)

type Reconciler struct {
	db                     *sql.DB
	runner                 *adaptiveRunner
	onProjectionReady      func()
	onRecoveryAlertCreated func()
	// processMu keeps one bounded OPEN-case scan coherent across the repeated
	// ProcessOne calls made by DrainProcessOne. scanCutoff is captured from the
	// database clock; unresolved rows move past it so each row is examined at
	// most once per drain without blocking later rows or forming a hot loop.
	processMu  sync.Mutex
	scanCutoff time.Time
}

// SetOnRecoveryAlertCreated registers a wake emitted only after a provider
// recovery summary commits durably without creating projection work.
func (r *Reconciler) SetOnRecoveryAlertCreated(fn func()) {
	if r != nil {
		r.onRecoveryAlertCreated = fn
	}
}

// SetOnProjectionReady registers a wake emitted only after a reconciled
// routing command and its actions commit durably.
func (r *Reconciler) SetOnProjectionReady(fn func()) {
	if r == nil {
		return
	}
	r.onProjectionReady = fn
}

// Notify wakes reconciliation after routing commits a newer source for an
// existing OPEN case. This keeps terminal status transitions out of the
// shared MaxIdle maintenance path while PostgreSQL remains the source of truth.
func (r *Reconciler) Notify() bool {
	if r == nil || r.runner == nil {
		return false
	}
	return r.runner.Notify()
}

type openCase struct {
	ID                 int64
	RoutingIdentityKey string
	NetworkFamily      string
	Version            int
	ReasonCode         ReasonCode
	EventType          string
	RawPayload         []byte
}

func NewReconciler(db *sql.DB) (*Reconciler, error) {
	if db == nil {
		return nil, fmt.Errorf("fund routing reconciliation database is required")
	}
	r := &Reconciler{db: db}
	r.runner = newAdaptiveRunner("fund routing OPEN-case reconciler", 30*time.Second, adaptiveschedule.DefaultMaxIdle, r.ProcessOne)
	return r, nil
}

func (r *Reconciler) ProcessOne(ctx context.Context) (_ bool, err error) {
	r.processMu.Lock()
	defer r.processMu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			r.scanCutoff = time.Time{}
		}
	}()
	if r.scanCutoff.IsZero() {
		if err = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&r.scanCutoff); err != nil {
			return false, fmt.Errorf("load OPEN routing scan cutoff: %w", err)
		}
	}
	var current openCase
	err = tx.QueryRowContext(ctx, `
SELECT routing.id, routing.routing_identity_key, routing.network_family, routing.version, routing.reason_code,
       webhook.event_type, webhook.raw_payload
FROM safeheron_transaction_routing_cases routing
JOIN LATERAL (
  SELECT source.safeheron_webhook_event_id
  FROM safeheron_transaction_routing_case_sources source
  WHERE source.case_id=routing.id
  ORDER BY source.provider_status_rank DESC, source.id DESC LIMIT 1
) source ON true
JOIN safeheron_webhook_events webhook ON webhook.id=source.safeheron_webhook_event_id
WHERE routing.decision='OPEN' AND routing.pending_command_id IS NULL
  AND routing.reason_code IN ('OWNERSHIP_UNKNOWN','CUSTOMER_POOL_UNASSIGNED','COMPANY_ACCOUNT_DISABLED','BEFORE_COMPANY_MONITORING','STATUS_NOT_TERMINAL')
  AND routing.updated_at < $1
ORDER BY routing.updated_at, routing.id
FOR UPDATE OF routing SKIP LOCKED
LIMIT 1`, r.scanCutoff).Scan(&current.ID, &current.RoutingIdentityKey, &current.NetworkFamily,
		&current.Version, &current.ReasonCode, &current.EventType, &current.RawPayload)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		r.scanCutoff = time.Time{}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var envelope struct {
		EventType   string                        `json:"eventType"`
		EventDetail safeheron.TransactionSnapshot `json:"eventDetail"`
	}
	if err = json.Unmarshal(current.RawPayload, &envelope); err != nil {
		return false, fmt.Errorf("decode OPEN routing source: %w", err)
	}
	candidates, err := BuildCandidates(envelope.EventDetail, current.NetworkFamily)
	if err != nil {
		return false, err
	}
	candidate, found := findCandidate(candidates, current.RoutingIdentityKey)
	if !found {
		return false, fmt.Errorf("OPEN routing case %d identity is absent from source", current.ID)
	}
	sourceOwner, err := lookupOwnership(ctx, tx, candidate.NetworkFamily, candidate.Occurrence.NormalizedSource)
	if err != nil {
		return false, err
	}
	destinationOwner, err := lookupOwnership(ctx, tx, candidate.NetworkFamily, candidate.Occurrence.NormalizedDestination)
	if err != nil {
		return false, err
	}
	eventType := current.EventType
	if envelope.EventType != "" {
		eventType = envelope.EventType
	}
	decision := Decide(DecisionInput{
		Candidate: candidate, SourceOwner: sourceOwner, DestinationOwner: destinationOwner,
		CustomerAdmission: AutomaticCustomerAdmission(eventType, envelope.EventDetail),
	})
	if decision.Decision == DecisionOpen {
		recoveryCreated := false
		_, err = tx.ExecContext(ctx, `UPDATE safeheron_transaction_routing_cases
SET reason_code=$2, updated_at=clock_timestamp() WHERE id=$1 AND decision='OPEN' AND pending_command_id IS NULL`, current.ID, string(decision.Reason))
		if err != nil {
			return false, err
		}
		if current.ReasonCode == ReasonStatusNotTerminal && decision.Reason != ReasonStatusNotTerminal {
			recoveryCreated, err = insertStatusRecoveryAlert(
				ctx, tx, current, candidate, decision, envelope.EventDetail, current.Version,
			)
			if err != nil {
				return false, err
			}
		}
		if err = tx.Commit(); err != nil {
			return false, err
		}
		if recoveryCreated && r.onRecoveryAlertCreated != nil {
			r.onRecoveryAlertCreated()
		}
		return true, nil
	}
	if err = reserveReconciledProjection(ctx, tx, current, candidate, decision, envelope.EventDetail); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	if r.onProjectionReady != nil {
		r.onProjectionReady()
	}
	return true, nil
}

func reserveReconciledProjection(
	ctx context.Context,
	tx *sql.Tx,
	current openCase,
	candidate Candidate,
	decision DecisionResult,
	snapshot safeheron.TransactionSnapshot,
) error {
	var commandID int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO safeheron_transaction_routing_case_commands
  (case_id, command_type, initiator, actor_scope, reason, idempotency_key, request_digest, expected_case_version)
VALUES ($1,'AUTO_ROUTE','SYSTEM','system:reconcile',$2,$3,$4,$5)
RETURNING id`, current.ID, string(decision.Reason), fmt.Sprintf("reconcile:%s:%d", candidate.RoutingIdentityKey, current.Version), routingRequestDigest(candidate, decision), current.Version).Scan(&commandID); err != nil {
		return err
	}
	if decision.RequiresCustomerProjection {
		if _, err := tx.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_case_actions
  (command_id,action_type,projection_kind,target_user_id) VALUES ($1,'APPLY_CUSTOMER','CUSTOMER',$2)`, commandID, decision.CustomerUserID); err != nil {
			return err
		}
	}
	if decision.RequiresCompanyProjection {
		if _, err := tx.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_case_actions
  (command_id,action_type,projection_kind,target_company_fund_account_id) VALUES ($1,'APPLY_COMPANY','COMPANY',$2)`, commandID, decision.CompanyFundAccountID); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE safeheron_transaction_routing_cases
SET decision=$2, reason_code=$3, requires_customer_projection=$4, requires_company_projection=$5,
    customer_user_id=$6, company_fund_account_id=$7, pending_command_id=$8,
    version=version+1, updated_at=now()
WHERE id=$1 AND version=$9 AND decision='OPEN' AND pending_command_id IS NULL`,
		current.ID, string(decision.Decision), string(decision.Reason), decision.RequiresCustomerProjection,
		decision.RequiresCompanyProjection, decision.CustomerUserID, decision.CompanyFundAccountID, commandID, current.Version)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return fmt.Errorf("reserve reconciled routing projection affected %d rows", rows)
	}
	if current.ReasonCode != ReasonStatusNotTerminal {
		return nil
	}
	_, err = insertStatusRecoveryAlert(ctx, tx, current, candidate, decision, snapshot, current.Version+1)
	return err
}

func insertStatusRecoveryAlert(
	ctx context.Context,
	tx *sql.Tx,
	current openCase,
	candidate Candidate,
	decision DecisionResult,
	snapshot safeheron.TransactionSnapshot,
	version int,
) (bool, error) {
	result, err := tx.ExecContext(ctx, statusRecoveryAlertSQL(),
		current.ID, version, string(decision.Decision), string(decision.Reason),
		candidate.SafeheronTxKey, candidate.RawCoinKey, candidate.NetworkFamily,
		candidate.Direction, candidate.Occurrence.Amount.String(),
		candidate.Occurrence.NormalizedSource, candidate.Occurrence.NormalizedDestination,
		strings.ToUpper(strings.TrimSpace(snapshot.TransactionStatus)), strings.TrimSpace(snapshot.TxHash),
		string(candidate.Occurrence.Occurrence.Input.MovementKind), strings.TrimSpace(snapshot.TransactionSubStatus),
	)
	if err != nil {
		return false, fmt.Errorf("insert Safeheron provider-status recovery alert: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect Safeheron provider-status recovery alert: %w", err)
	}
	return rows == 1, nil
}

func statusRecoveryAlertSQL() string {
	return `INSERT INTO safeheron_transaction_routing_alerts
  (case_id,alert_type,transition_key,severity,payload)
SELECT $1,'RECOVERY_SUMMARY','sla:recovered:version:' || $2::integer::text,'INFO',
       jsonb_strip_nulls(jsonb_build_object(
         'environment',COALESCE((
           SELECT prior.payload->>'environment'
           FROM safeheron_transaction_routing_alerts prior
           WHERE prior.case_id=$1 AND prior.alert_type='SLA_ESCALATION'
             AND prior.transition_key LIKE 'sla:onchain:level:%'
           ORDER BY prior.created_at DESC,prior.id DESC LIMIT 1
         ),'unknown'),
         'case_id',$1::bigint,
         'sla_scope','ONCHAIN',
         'resolved_decision',$3::text,
         'resolved_reason_code',$4::text,
         'safeheron_tx_key',$5::text,
         'raw_coin_key',$6::text,
         'network_family',$7::text,
         'direction',$8::text,
         'amount',$9::text,
         'source_address',$10::text,
         'destination_address',$11::text,
         'transaction_status',$12::text,
         'tx_hash',NULLIF($13::text,''),
         'movement_kind',$14::text,
         'transaction_sub_status',NULLIF($15::text,''),
         'effective_event_time',routing.effective_event_time,
         'sla_started_at',chain_progress.started_at,
         'stuck_seconds',floor(extract(epoch FROM now()-chain_progress.started_at))::bigint,
         'last_source_event_type',webhook.event_type,
         'last_source_received_at',webhook.received_at AT TIME ZONE 'UTC',
         'last_api_checked_at',status_check.last_checked_at,
         'last_api_check_outcome',status_check.last_check_outcome,
         'last_api_error_code',status_check.last_error_code,
         'resolved_at',now()
       ))
FROM safeheron_transaction_routing_cases routing
JOIN LATERAL (
  SELECT linked.safeheron_webhook_event_id
  FROM safeheron_transaction_routing_case_sources linked
  WHERE linked.case_id=routing.id
  ORDER BY linked.provider_status_rank DESC,linked.id DESC LIMIT 1
) source ON true
JOIN safeheron_webhook_events webhook ON webhook.id=source.safeheron_webhook_event_id
LEFT JOIN safeheron_transaction_routing_status_checks status_check
  ON status_check.safeheron_tx_key=routing.safeheron_tx_key
JOIN LATERAL (
  SELECT min(linked.linked_at) AS started_at
  FROM safeheron_transaction_routing_case_sources linked
  WHERE linked.case_id=routing.id
    AND upper(linked.provider_status) IN ('BROADCASTING','CONFIRMING')
) chain_progress ON chain_progress.started_at IS NOT NULL
WHERE routing.id=$1 AND EXISTS (
  SELECT 1 FROM safeheron_transaction_routing_alerts prior
  WHERE prior.case_id=$1 AND prior.alert_type='SLA_ESCALATION'
    AND prior.payload->>'reason_code'='STATUS_NOT_TERMINAL'
    AND prior.transition_key LIKE 'sla:onchain:level:%'
)
ON CONFLICT (case_id,alert_type,transition_key) DO NOTHING`
}

func findCandidate(candidates []Candidate, identity string) (Candidate, bool) {
	for _, candidate := range candidates {
		if candidate.RoutingIdentityKey == identity {
			return candidate, true
		}
	}
	return Candidate{}, false
}

func (r *Reconciler) Run(ctx context.Context) {
	if r == nil || r.runner == nil {
		return
	}
	r.runner.Run(ctx)
}
