package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDispatcherPropagatesCancellationIntoHandler(t *testing.T) {
	dispatcher := NewDispatcher()
	started := make(chan context.Context, 1)
	release := make(chan struct{})
	dispatcher.Register("recover", func(ctx context.Context, id string) error {
		started <- ctx
		<-release
		return errors.New("publisher unavailable")
	})
	if !dispatcher.Enqueue("recover", "lot-29") {
		t.Fatal("enqueue failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(ctx) }()
	handlerCtx := <-started
	cancel()
	select {
	case <-handlerCtx.Done():
	case <-time.After(100 * time.Millisecond):
		close(release)
		dispatcher.queue.Close()
		t.Fatal("dispatcher cancellation did not reach handler")
	}
	close(release)
	dispatcher.queue.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not stop")
	}
}
