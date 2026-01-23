// internal/scheduler/metrics.go
package scheduler

import (
	"sync"
	"time"
)

type SchedulerMetrics struct {
	mu sync.RWMutex

	InterestRunCount      int64
	InterestSuccessCount  int64
	InterestErrorCount    int64
	TotalOrdersProcessed  int64
	TotalInterestAccrued  float64
	LastRunTime           time.Time
	LastSuccessTime       time.Time
	LastErrorTime         time.Time
	LastErrorMessage      string
	AverageOrdersPerRun   float64
	AverageInterestPerRun float64
}

func NewSchedulerMetrics() *SchedulerMetrics {
	return &SchedulerMetrics{}
}

func (m *SchedulerMetrics) RecordInterestRun(success bool, ordersProcessed int, interestAccrued float64, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.InterestRunCount++
	m.TotalOrdersProcessed += int64(ordersProcessed)
	m.TotalInterestAccrued += interestAccrued
	m.LastRunTime = time.Now()

	if success {
		m.InterestSuccessCount++
		m.LastSuccessTime = time.Now()
	} else {
		m.InterestErrorCount++
		m.LastErrorTime = time.Now()
		m.LastErrorMessage = errorMsg
	}

	m.updateAverages()
}

func (m *SchedulerMetrics) updateAverages() {
	if m.InterestRunCount == 0 {
		return
	}
	m.AverageOrdersPerRun = float64(m.TotalOrdersProcessed) / float64(m.InterestRunCount)
	m.AverageInterestPerRun = m.TotalInterestAccrued / float64(m.InterestRunCount)
}

func (m *SchedulerMetrics) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"interest_run_count":       m.InterestRunCount,
		"interest_success_count":   m.InterestSuccessCount,
		"interest_error_count":     m.InterestErrorCount,
		"success_rate":             m.getSuccessRate(),
		"total_orders_processed":   m.TotalOrdersProcessed,
		"total_interest_accrued":   m.TotalInterestAccrued,
		"average_orders_per_run":   m.AverageOrdersPerRun,
		"average_interest_per_run": m.AverageInterestPerRun,
		"last_run_time":            m.LastRunTime.Format(time.RFC3339),
		"last_success_time":        m.formatTime(m.LastSuccessTime),
		"last_error_time":          m.formatTime(m.LastErrorTime),
		"last_error_message":       m.LastErrorMessage,
	}
}

func (m *SchedulerMetrics) getSuccessRate() float64 {
	if m.InterestRunCount == 0 {
		return 0
	}
	return float64(m.InterestSuccessCount) / float64(m.InterestRunCount) * 100
}

func (m *SchedulerMetrics) formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format(time.RFC3339)
}

func (m *SchedulerMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.InterestRunCount = 0
	m.InterestSuccessCount = 0
	m.InterestErrorCount = 0
	m.TotalOrdersProcessed = 0
	m.TotalInterestAccrued = 0
	m.LastRunTime = time.Time{}
	m.LastSuccessTime = time.Time{}
	m.LastErrorTime = time.Time{}
	m.LastErrorMessage = ""
	m.AverageOrdersPerRun = 0
	m.AverageInterestPerRun = 0
}
