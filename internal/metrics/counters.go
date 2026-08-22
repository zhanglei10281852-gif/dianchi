package metrics

import "sync"

type Counters struct {
	mu     sync.RWMutex
	values map[string]int64
}

func New() *Counters                         { return &Counters{values: map[string]int64{}} }
func (c *Counters) Inc(name string)          { c.mu.Lock(); defer c.mu.Unlock(); c.values[name]++ }
func (c *Counters) Add(name string, n int64) { c.mu.Lock(); defer c.mu.Unlock(); c.values[name] += n }
func (c *Counters) Get(name string) int64    { c.mu.RLock(); defer c.mu.RUnlock(); return c.values[name] }
func (c *Counters) Snapshot() map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := map[string]int64{}
	for k, v := range c.values {
		out[k] = v
	}
	return out
}
func (c *Counters) Reset() { c.mu.Lock(); defer c.mu.Unlock(); c.values = map[string]int64{} }
