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
		attempt := r.Attempts[key]
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
		if err := r.sleep(ctx, r.Policy.Delay(attempt)); err != nil {
			return err
		}
	}
}

// sleep blocks for d unless ctx is canceled or expires first. Unlike time.Sleep,
// it reports ctx's error promptly so a caller that cancels during the backoff
// between attempts does not have to wait for the full delay to elapse.
func (r *Retrier) sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
func IsPermanent(err error) bool { return errors.Is(err, context.Canceled) }
