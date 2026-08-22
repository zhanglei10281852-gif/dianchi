package service

import (
	"context"
	"sync"
	"time"

	retention "github.com/zhanglei10281852-gif/dianchi/internal/audit"
)

type RetentionRecord struct {
	ID, Tenant, Kind string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	Payload          []byte
}
type RetentionStore struct {
	mu   sync.RWMutex
	rows map[string]RetentionRecord
}

func NewRetentionStore() *RetentionStore { return &RetentionStore{rows: map[string]RetentionRecord{}} }
func (r *RetentionStore) Put(ctx context.Context, v RetentionRecord) error {
	v.Payload = retention.CopyPayload(v.Payload)
	r.mu.Lock()
	r.rows[v.ID] = v
	r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
func (r *RetentionStore) Get(ctx context.Context, id string) (RetentionRecord, bool) {
	if err := ctx.Err(); err != nil {
		return RetentionRecord{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.rows[id]
	v.Payload = append([]byte(nil), v.Payload...)
	return v, ok
}
func (r *RetentionStore) Expire(now time.Time) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for id, v := range r.rows {
		if !now.Before(v.ExpiresAt) {
			delete(r.rows, id)
			n++
		}
	}
	return n
}
func (r *RetentionStore) Count() int { r.mu.RLock(); defer r.mu.RUnlock(); return len(r.rows) }
