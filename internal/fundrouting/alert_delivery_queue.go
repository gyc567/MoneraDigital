package fundrouting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"monera-digital/internal/alert"
)

type postgresRoutingAlertQueue struct {
	db *sql.DB
}

func newPostgresRoutingAlertQueue(db *sql.DB) (*postgresRoutingAlertQueue, error) {
	if db == nil {
		return nil, fmt.Errorf("routing alert database is required")
	}
	return &postgresRoutingAlertQueue{db: db}, nil
}

func (q *postgresRoutingAlertQueue) NextDue(ctx context.Context) (time.Time, error) {
	var due sql.NullTime
	err := q.db.QueryRowContext(ctx, `SELECT min(due_at) FROM (
  SELECT next_attempt_at AS due_at
  FROM safeheron_transaction_routing_alert_deliveries
  WHERE status='PENDING' AND next_attempt_at > now()
  UNION ALL
  SELECT next_attempt_at AS due_at
  FROM safeheron_transaction_routing_alert_deliveries
  WHERE status='FAILED_DEFINITE' AND next_attempt_at > now()
  UNION ALL
  SELECT lease_expires_at AS due_at
  FROM safeheron_transaction_routing_alert_deliveries
  WHERE status='DISPATCHING' AND lease_expires_at > now()
) deadlines`).Scan(&due)
	if err != nil || !due.Valid {
		return time.Time{}, err
	}
	return due.Time, nil
}

func (q *postgresRoutingAlertQueue) EnsureDeliveries(
	ctx context.Context,
	sinks []alert.RoutingSink,
	larkWindow time.Duration,
) (created bool, err error) {
	if len(sinks) == 0 {
		return false, nil
	}
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	var alertID int64
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT alert.id,alert.created_at
FROM safeheron_transaction_routing_alerts alert
WHERE NOT EXISTS (
  SELECT 1 FROM safeheron_transaction_routing_alert_deliveries delivery WHERE delivery.alert_id=alert.id
)
	ORDER BY alert.id FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&alertID, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, sink := range sinks {
		var nextAttempt any
		if sink.Kind == "LARK" {
			nextAttempt = larkRoutingBatchDue(createdAt, larkWindow)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO safeheron_transaction_routing_alert_deliveries
  (alert_id,sink_kind,recipient_fingerprint,next_attempt_at) VALUES ($1,$2,$3,$4)
ON CONFLICT (alert_id,sink_kind,recipient_fingerprint) DO NOTHING`,
			alertID, sink.Kind, sink.Fingerprint, nextAttempt); err != nil {
			return false, err
		}
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

type routingAlertClaimSeed struct {
	ID                    int64
	SinkKind              string
	Fingerprint           string
	Status                string
	AutomaticAttemptCount int
	NextAttemptAt         sql.NullTime
	AlertType             string
	Severity              string
	AlertCreatedAt        time.Time
	ManualReplay          bool
}

func (q *postgresRoutingAlertQueue) Claim(
	ctx context.Context,
	workerID string,
	batchLimit int,
) (_ []claimedDelivery, err error) {
	if batchLimit <= 0 {
		batchLimit = 1
	}
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	var seed routingAlertClaimSeed
	err = tx.QueryRowContext(ctx, `SELECT delivery.id,delivery.sink_kind,delivery.recipient_fingerprint,
       delivery.status,delivery.automatic_attempt_count,delivery.next_attempt_at,
	       alert.alert_type,alert.severity,alert.created_at,
       EXISTS (
         SELECT 1 FROM safeheron_transaction_routing_alert_delivery_attempts attempt
         WHERE attempt.delivery_id=delivery.id AND attempt.attempt_kind='MANUAL_REPLAY'
           AND attempt.outcome='IN_PROGRESS'
       ) AS manual_replay
FROM safeheron_transaction_routing_alert_deliveries delivery
JOIN safeheron_transaction_routing_alerts alert ON alert.id=delivery.alert_id
WHERE (delivery.status='PENDING' AND (delivery.next_attempt_at IS NULL OR delivery.next_attempt_at<=now()))
   OR (delivery.status='FAILED_DEFINITE' AND delivery.next_attempt_at IS NOT NULL AND delivery.next_attempt_at<=now())
ORDER BY COALESCE(delivery.next_attempt_at,delivery.created_at),delivery.id
FOR UPDATE OF delivery SKIP LOCKED LIMIT 1`).Scan(
		&seed.ID, &seed.SinkKind, &seed.Fingerprint, &seed.Status,
		&seed.AutomaticAttemptCount, &seed.NextAttemptAt, &seed.AlertType,
		&seed.Severity, &seed.AlertCreatedAt, &seed.ManualReplay,
	)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	ids := []int64{seed.ID}
	batchableInitial := seed.Status == "PENDING" && seed.AutomaticAttemptCount == 0
	batchableRetry := seed.Status == "FAILED_DEFINITE"
	if seed.SinkKind == "LARK" && (batchableInitial || batchableRetry) && seed.NextAttemptAt.Valid && !seed.ManualReplay {
		ids, err = q.claimableLarkBatchIDs(ctx, tx, seed, batchLimit)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			ids = []int64{seed.ID}
		}
	}

	batch := make([]claimedDelivery, 0, len(ids))
	for _, deliveryID := range ids {
		delivery, claimErr := q.claimOne(ctx, tx, workerID, deliveryID)
		if claimErr != nil {
			return nil, claimErr
		}
		batch = append(batch, delivery)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return batch, nil
}

func (q *postgresRoutingAlertQueue) claimableLarkBatchIDs(
	ctx context.Context,
	tx *sql.Tx,
	seed routingAlertClaimSeed,
	batchLimit int,
) ([]int64, error) {
	windowStart := seed.AlertCreatedAt.UTC().Truncate(routingLarkBatchWindow)
	windowEnd := windowStart.Add(routingLarkBatchWindow)
	rows, err := tx.QueryContext(ctx, claimableLarkBatchQuery(),
		seed.Fingerprint, seed.NextAttemptAt.Time, seed.AlertType,
		windowStart, windowEnd, seed.Status, batchLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0, batchLimit)
	payloadBytes := 0
	for rows.Next() {
		var id int64
		var size int
		if err := rows.Scan(&id, &size); err != nil {
			return nil, err
		}
		if len(ids) > 0 && payloadBytes+size > routingLarkBatchMaxPayloadBytes {
			break
		}
		ids = append(ids, id)
		payloadBytes += size
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func claimableLarkBatchQuery() string {
	return `SELECT delivery.id,octet_length(alert.payload::text)
FROM safeheron_transaction_routing_alert_deliveries delivery
JOIN safeheron_transaction_routing_alerts alert ON alert.id=delivery.alert_id
WHERE delivery.sink_kind='LARK' AND delivery.recipient_fingerprint=$1
	  AND delivery.next_attempt_at=$2 AND delivery.next_attempt_at<=now()
	  AND alert.alert_type=$3
	  AND alert.created_at >= $4 AND alert.created_at < $5
	  AND (
	    ($6='PENDING' AND delivery.status='PENDING' AND delivery.automatic_attempt_count=0
	      AND NOT EXISTS (
	        SELECT 1 FROM safeheron_transaction_routing_alert_delivery_attempts attempt
	        WHERE attempt.delivery_id=delivery.id
	      ))
	    OR
	    ($6='FAILED_DEFINITE' AND delivery.status='FAILED_DEFINITE'
	      AND NOT EXISTS (
	        SELECT 1 FROM safeheron_transaction_routing_alert_delivery_attempts attempt
	        WHERE attempt.delivery_id=delivery.id AND attempt.attempt_kind='MANUAL_REPLAY'
	          AND attempt.outcome='IN_PROGRESS'
	      ))
	  )
ORDER BY delivery.id
FOR UPDATE OF delivery SKIP LOCKED LIMIT $7`
}

func (q *postgresRoutingAlertQueue) claimOne(
	ctx context.Context,
	tx *sql.Tx,
	workerID string,
	deliveryID int64,
) (claimedDelivery, error) {
	delivery := claimedDelivery{ID: deliveryID}
	result, err := tx.ExecContext(ctx, `UPDATE safeheron_transaction_routing_alert_deliveries
SET status='DISPATCHING',lease_owner=$2,lease_expires_at=now()+interval '30 seconds',
    next_attempt_at=NULL,updated_at=now()
WHERE id=$1`, deliveryID, workerID)
	if err != nil {
		return claimedDelivery{}, err
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr != nil || rows != 1 {
		return claimedDelivery{}, fmt.Errorf("routing alert delivery %d could not be leased", deliveryID)
	}
	err = tx.QueryRowContext(ctx, `SELECT id,attempt_number
FROM safeheron_transaction_routing_alert_delivery_attempts
WHERE delivery_id=$1 AND attempt_kind='MANUAL_REPLAY' AND outcome='IN_PROGRESS'
ORDER BY attempt_number DESC LIMIT 1 FOR UPDATE`, delivery.ID).Scan(&delivery.AttemptID, &delivery.Attempt)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `UPDATE safeheron_transaction_routing_alert_deliveries
SET automatic_attempt_count=automatic_attempt_count+1 WHERE id=$1
RETURNING automatic_attempt_count`, delivery.ID).Scan(&delivery.AutomaticAttemptCount)
		if err != nil {
			return claimedDelivery{}, err
		}
		err = tx.QueryRowContext(ctx, `INSERT INTO safeheron_transaction_routing_alert_delivery_attempts
  (delivery_id,attempt_number,attempt_kind,outcome)
SELECT $1,COALESCE(max(attempt_number),0)+1,'AUTO','IN_PROGRESS'
FROM safeheron_transaction_routing_alert_delivery_attempts WHERE delivery_id=$1
RETURNING id,attempt_number`, delivery.ID).Scan(&delivery.AttemptID, &delivery.Attempt)
	} else if err == nil {
		err = tx.QueryRowContext(ctx, `SELECT automatic_attempt_count
FROM safeheron_transaction_routing_alert_deliveries WHERE id=$1 FOR UPDATE`, delivery.ID).
			Scan(&delivery.AutomaticAttemptCount)
	}
	if err != nil {
		return claimedDelivery{}, err
	}
	err = tx.QueryRowContext(ctx, `SELECT delivery.alert_id,alert.case_id,
  delivery.sink_kind,delivery.recipient_fingerprint,
  alert.severity,alert.alert_type,alert.payload,alert.created_at
FROM safeheron_transaction_routing_alert_deliveries delivery
JOIN safeheron_transaction_routing_alerts alert ON alert.id=delivery.alert_id
WHERE delivery.id=$1 AND delivery.lease_owner=$2`, delivery.ID, workerID).Scan(
		&delivery.AlertID, &delivery.CaseID, &delivery.SinkKind, &delivery.Fingerprint,
		&delivery.Severity, &delivery.AlertType, &delivery.Payload, &delivery.AlertCreatedAt,
	)
	if err != nil {
		return claimedDelivery{}, err
	}
	return delivery, nil
}

func (q *postgresRoutingAlertQueue) Finish(
	ctx context.Context,
	workerID string,
	batch []claimedDelivery,
	outcome alert.RoutingDeliveryOutcome,
) (err error) {
	if len(batch) == 0 {
		return nil
	}
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	for _, delivery := range batch {
		status, attemptOutcome, completed, nextAttempt := "AMBIGUOUS", "DELIVERY_UNKNOWN", true, false
		switch outcome {
		case alert.RoutingDeliverySent:
			status, attemptOutcome = "SENT", "SENT"
		case alert.RoutingDeliveryDefinitelyNotSent:
			status, attemptOutcome, completed = "FAILED_DEFINITE", "DEFINITELY_NOT_SENT", false
			nextAttempt = delivery.AutomaticAttemptCount < 3
		}
		if _, err = tx.ExecContext(ctx, `UPDATE safeheron_transaction_routing_alert_delivery_attempts
SET outcome=$2,finished_at=now() WHERE id=$1 AND outcome='IN_PROGRESS'`, delivery.AttemptID, attemptOutcome); err != nil {
			return err
		}
		result, updateErr := tx.ExecContext(ctx, `UPDATE safeheron_transaction_routing_alert_deliveries
SET status=$2,lease_owner=NULL,lease_expires_at=NULL,updated_at=now(),
    completed_at=CASE WHEN $3 THEN now() ELSE NULL END,
    next_attempt_at=CASE WHEN $4 THEN now()+interval '30 seconds' ELSE NULL END
WHERE id=$1 AND lease_owner=$5`, delivery.ID, status, completed, nextAttempt, workerID)
		if updateErr != nil {
			return updateErr
		}
		rows, rowsErr := result.RowsAffected()
		if rowsErr != nil || rows != 1 {
			return fmt.Errorf("routing alert delivery %d lease lost", delivery.ID)
		}
	}
	return tx.Commit()
}

func (q *postgresRoutingAlertQueue) SweepExpired(ctx context.Context) (err error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `UPDATE safeheron_transaction_routing_alert_delivery_attempts attempt
SET outcome='DELIVERY_UNKNOWN',finished_at=now()
FROM safeheron_transaction_routing_alert_deliveries delivery
WHERE delivery.id=attempt.delivery_id AND delivery.status='DISPATCHING'
  AND delivery.lease_expires_at<=now() AND attempt.outcome='IN_PROGRESS'`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE safeheron_transaction_routing_alert_deliveries
SET status='AMBIGUOUS',lease_owner=NULL,lease_expires_at=NULL,completed_at=now(),updated_at=now()
WHERE status='DISPATCHING' AND lease_expires_at<=now()`); err != nil {
		return err
	}
	return tx.Commit()
}

var _ routingAlertQueue = (*postgresRoutingAlertQueue)(nil)
