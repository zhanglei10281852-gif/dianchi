package metrics

import (
	"sort"
	"sync"
	"time"
)

type Histogram struct {
	mu     sync.Mutex
	values map[string][]time.Duration
}

func NewHistogram() *Histogram { return &Histogram{values: map[string][]time.Duration{}} }
func (h *Histogram) Observe(name string, d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values[name] = append(h.values[name], d)
}
func (h *Histogram) Percentile(name string, p float64) time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	v := append([]time.Duration(nil), h.values[name]...)
	if len(v) == 0 {
		return 0
	}
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	return v[int(float64(len(v)-1)*p)]
}
func (h *Histogram) Count(name string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.values[name])
}
