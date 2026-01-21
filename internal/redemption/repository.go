package redemption

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type RedemptionRepository interface {
	Create(record *RedemptionRecord) error
	Get(id string) (*RedemptionRecord, error)
	Update(record *RedemptionRecord) error
	List() ([]*RedemptionRecord, error)
}

type InMemoryRedemptionRepository struct {
	mu     sync.RWMutex
	data   map[string]*RedemptionRecord
	nextID int64
}

func NewInMemoryRedemptionRepository() *InMemoryRedemptionRepository {
	return &InMemoryRedemptionRepository{
		data:   make(map[string]*RedemptionRecord),
		nextID: 0,
	}
}

func (r *InMemoryRedemptionRepository) Create(record *RedemptionRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	newID := atomic.AddInt64(&r.nextID, 1)
	record.ID = fmt.Sprintf("REDEEM-%d-%d", time.Now().UnixNano(), newID)
	r.data[record.ID] = record
	return nil
}

func (r *InMemoryRedemptionRepository) Get(id string) (*RedemptionRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.data[id]
	if !ok {
		return nil, fmt.Errorf("redemption not found")
	}
	return rec, nil
}

func (r *InMemoryRedemptionRepository) Update(record *RedemptionRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[record.ID]; !ok {
		return fmt.Errorf("redemption not found")
	}
	r.data[record.ID] = record
	return nil
}

func (r *InMemoryRedemptionRepository) List() ([]*RedemptionRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*RedemptionRecord, 0, len(r.data))
	for _, v := range r.data {
		out = append(out, v)
	}
	return out, nil
}
