package worker

import (
	"context"
	"sync"
	"time"
)

type Health struct {
	mu       sync.RWMutex
	lastRun  time.Time
	failures int
	running  bool
}

type HealthSnapshot struct {
	LastRun  time.Time
	Failures int
	Running  bool
}

func (h *Health) Start() { h.mu.Lock(); defer h.mu.Unlock(); h.running = true }
func (h *Health) Stop()  { h.mu.Lock(); defer h.mu.Unlock(); h.running = false }
func (h *Health) Record(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastRun = time.Now().UTC()
	if err != nil {
		h.failures++
	}
}
func (h *Health) Snapshot() (time.Time, int, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastRun, h.failures, h.running
}
func (h *Health) State() HealthSnapshot {
	lastRun, failures, running := h.Snapshot()
	return HealthSnapshot{LastRun: lastRun, Failures: failures, Running: running}
}
func Wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
