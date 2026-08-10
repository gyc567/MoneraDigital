package fundrouting

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"monera-digital/internal/adaptiveschedule"
	"monera-digital/internal/alert"
)

const (
	routingLarkBatchWindow = 5 * time.Minute
	routingLarkBatchLimit  = 20
	// Payload bytes are capped below Lark's final rendered-message boundary;
	// full txKey and address fields remain intact in every selected alert.
	routingLarkBatchMaxPayloadBytes = 8 * 1024
	routingFanoutLimit              = 100
)

type RoutingAlertSender interface {
	RoutingSinks() []alert.RoutingSink
	SendRouting(context.Context, string, string, string, string, map[string]string) alert.RoutingDeliveryOutcome
}

type routingAlertQueue interface {
	NextDue(context.Context) (time.Time, error)
	SweepExpired(context.Context) error
	EnsureDeliveries(context.Context, []alert.RoutingSink, time.Duration) (bool, error)
	Claim(context.Context, string, int) ([]claimedDelivery, error)
	Finish(context.Context, string, []claimedDelivery, alert.RoutingDeliveryOutcome) error
}

type AlertNotifier struct {
	queue    routingAlertQueue
	sender   RoutingAlertSender
	workerID string
	runner   *adaptiveRunner
}

type claimedDelivery struct {
	ID                    int64
	AlertID               int64
	CaseID                int64
	AttemptID             int64
	Attempt               int
	AutomaticAttemptCount int
	SinkKind              string
	Fingerprint           string
	Severity              string
	AlertType             string
	Payload               []byte
	AlertCreatedAt        time.Time
}

func NewAlertNotifier(db *sql.DB, sender RoutingAlertSender) (*AlertNotifier, error) {
	if db == nil {
		return nil, fmt.Errorf("routing alert database is required")
	}
	queue, err := newPostgresRoutingAlertQueue(db)
	if err != nil {
		return nil, err
	}
	return newAlertNotifierWithQueue(queue, sender)
}

func newAlertNotifierWithQueue(queue routingAlertQueue, sender RoutingAlertSender) (*AlertNotifier, error) {
	if queue == nil || sender == nil {
		return nil, fmt.Errorf("routing alert queue and sender are required")
	}
	n := &AlertNotifier{queue: queue, sender: sender, workerID: newProjectionWorkerID()}
	n.runner = newAdaptiveRunner("fund routing alert notifier", time.Second, adaptiveschedule.DefaultMaxIdle, n.ProcessOne)
	n.runner.setNextDue(n.NextDue)
	return n, nil
}

// NextDue returns the earliest durable initial batch, delivery retry, or lease
// recovery deadline.
func (n *AlertNotifier) NextDue(ctx context.Context) (time.Time, error) {
	if n == nil || n.queue == nil {
		return time.Time{}, nil
	}
	return n.queue.NextDue(ctx)
}

// Notify wakes alert delivery after durable alert rows are written.
func (n *AlertNotifier) Notify() bool {
	if n == nil || n.runner == nil {
		return false
	}
	return n.runner.Notify()
}

func (n *AlertNotifier) ProcessOne(ctx context.Context) (bool, error) {
	if n == nil || n.queue == nil || n.sender == nil {
		return false, fmt.Errorf("routing alert notifier is not configured")
	}
	if err := n.queue.SweepExpired(ctx); err != nil {
		return false, err
	}
	materialized := false
	for range routingFanoutLimit {
		created, err := n.queue.EnsureDeliveries(ctx, n.sender.RoutingSinks(), routingLarkBatchWindow)
		if err != nil {
			return materialized, err
		}
		if !created {
			break
		}
		materialized = true
	}
	batch, err := n.queue.Claim(ctx, n.workerID, routingLarkBatchLimit)
	if err != nil {
		return materialized, err
	}
	if len(batch) == 0 {
		return materialized, nil
	}
	for _, delivery := range batch[1:] {
		if delivery.SinkKind != batch[0].SinkKind || delivery.Fingerprint != batch[0].Fingerprint {
			return true, fmt.Errorf("routing alert queue returned a mixed sink batch")
		}
	}
	severity, title, fields := routingAlertPresentation(batch)
	outcome := n.sender.SendRouting(
		ctx, batch[0].SinkKind, batch[0].Fingerprint, severity, title, fields,
	)
	return true, n.queue.Finish(ctx, n.workerID, batch, outcome)
}

func larkRoutingBatchDue(createdAt time.Time, window time.Duration) time.Time {
	if window <= 0 {
		window = routingLarkBatchWindow
	}
	return createdAt.UTC().Truncate(window).Add(window)
}

func (n *AlertNotifier) Run(ctx context.Context) {
	if n != nil && n.runner != nil {
		n.runner.Run(ctx)
	}
}
