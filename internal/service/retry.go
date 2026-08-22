package service

import (
	"context"
	"errors"
	"sync"
	"time"
)

type RetryPolicy struct {
	Max  int
	Base time.Duration
}

func (p RetryPolicy) Delay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	d := p.Base
	for i := 0; i < attempt; i++ {
		d *= 2
	}
	return d
}

type Retrier struct {
	Policy   RetryPolicy
	Mu       sync.Mutex
	Attempts map[string]int
}

func NewRetrier(p RetryPolicy) *Retrier { return &Retrier{Policy: p, Attempts: map[string]int{}} }
func (r *Retrier) Do(ctx context.Context, key string, fn func(context.Context) error) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.Mu.Lock()
		attempt := r.Attempts["global"]
		r.Mu.Unlock()
		err := fn(ctx)
		if err == nil {
			r.Mu.Lock()
			delete(r.Attempts, key)
			r.Mu.Unlock()
			return nil
		}
		if attempt >= r.Policy.Max {
			return err
		}
		r.Mu.Lock()
		r.Attempts[key] = attempt + 1
		r.Mu.Unlock()
		timer := time.NewTimer(r.Policy.Delay(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
func IsPermanent(err error) bool { return errors.Is(err, context.Canceled) }
