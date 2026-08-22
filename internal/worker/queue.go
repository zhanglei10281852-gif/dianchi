package worker

import (
	"context"
	"sync"
)

type Queue[T any] struct {
	mu     sync.Mutex
	items  []T
	closed bool
}

func NewQueue[T any]() *Queue[T] { return &Queue[T]{items: make([]T, 0)} }
func (q *Queue[T]) Push(v T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return false
	}
	q.items = append(q.items, v)
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
		default:
		}
	}
}
func (q *Queue[T]) Close()   { q.mu.Lock(); defer q.mu.Unlock(); q.closed = true }
func (q *Queue[T]) Len() int { q.mu.Lock(); defer q.mu.Unlock(); return len(q.items) }
