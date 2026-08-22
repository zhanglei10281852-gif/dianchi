package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrierCancellationInterruptsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	retrier := NewRetrier(RetryPolicy{Max: 3, Base: 2 * time.Second})
	started := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		done <- retrier.Do(ctx, "publisher-27", func(context.Context) error {
			started <- struct{}{}
			return errors.New("temporary outage")
		})
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("retry returned %v after cancellation", err)
		}
	case <-time.After(150 * time.Millisecond):
		t.Fatal("retry remained blocked in backoff after cancellation")
	}
}
