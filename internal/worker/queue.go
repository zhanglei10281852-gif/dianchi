package worker

import (
	"context"
	"sync"
)

type Queue[T any] struct {
	mu     sync.Mutex
	items  []T
	closed bool
	notify chan struct{}
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{items: make([]T, 0), notify: make(chan struct{}, 1)}
}
// HandlerContext derives the context passed to a registered handler.
// It must propagate cancellation from the dispatcher's Run context so that
// when the outer Run is cancelled (e.g. shutdown while a handler is invoking
// a downstream publisher), the handler observes ctx.Done and aborts its writes
// instead of continuing with a context that never expires.
func HandlerContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (q *Queue[T]) Push(v T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return false
	}
	q.items = append(q.items, v)
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return true
}
func (q *Queue[T]) Pop(ctx context.Context) (T, bool) {
	for {
		q.mu.Lock()
		if len(q.items) > 0 {
			v := q.items[0]
			q.items = q.items[1:]
			q.mu.Unlock()
			return v, true
		}
		closed := q.closed
		q.mu.Unlock()
		if closed {
			var zero T
			return zero, false
		}
		select {
		case <-ctx.Done():
			var zero T
			return zero, false
		case <-q.notify:
		}
	}
}
func (q *Queue[T]) Close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
}
func (q *Queue[T]) Len() int { q.mu.Lock(); defer q.mu.Unlock(); return len(q.items) }
