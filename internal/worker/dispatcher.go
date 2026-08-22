package worker

import (
	"context"
	"strings"
	"sync"
	"time"
)

type Handler func(context.Context, string) error
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	queue    *Queue[string]
	health   *Health
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: map[string]Handler{}, queue: NewQueue[string](), health: &Health{}}
}
func (d *Dispatcher) Register(kind string, h Handler) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.handlers[kind]; ok {
		return false
	}
	d.handlers[kind] = h
	return true
}
func (d *Dispatcher) Enqueue(kind, id string) bool { return d.queue.Push(kind + ":" + id) }
func (d *Dispatcher) Run(ctx context.Context) error {
	d.health.Start()
	defer d.health.Stop()
	for {
		item, ok := d.queue.Pop(ctx)
		if !ok {
			return ctx.Err()
		}
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			d.health.Record(ErrDuplicate)
			continue
		}
		d.mu.RLock()
		h := d.handlers[parts[0]]
		d.mu.RUnlock()
		if h == nil {
			d.health.Record(ErrDuplicate)
			continue
		}
		handlerCtx := HandlerContext(ctx)
		err := h(handlerCtx, parts[1])
		d.health.Record(err)
		if err != nil {
			if err := Wait(ctx, 10*time.Millisecond); err != nil {
				return err
			}
		}
	}
}
func (d *Dispatcher) Health() HealthSnapshot { return d.health.State() }
