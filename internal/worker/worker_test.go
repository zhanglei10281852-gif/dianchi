package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestQueuePopWakesForPushAndClose(t *testing.T) {
	q := NewQueue[string]()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	values := make(chan string, 1)
	go func() {
		value, ok := q.Pop(ctx)
		if ok {
			values <- value
		}
	}()
	if !q.Push("lot-1") {
		t.Fatal("push rejected")
	}
	select {
	case value := <-values:
		if value != "lot-1" {
			t.Fatal(value)
		}
	case <-ctx.Done():
		t.Fatal("pop did not wake")
	}
	q.Close()
	if _, ok := q.Pop(context.Background()); ok {
		t.Fatal("closed empty queue produced value")
	}
}

func TestQueueCancellationStopsBlockedPop(t *testing.T) {
	q := NewQueue[int]()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, ok := q.Pop(ctx); ok {
			t.Error("cancelled pop succeeded")
		}
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancelled pop remained blocked")
	}
}

func TestSchedulerCancellationDuringRunDoesNotResurrectJob(t *testing.T) {
	s := NewScheduler()
	started := make(chan struct{})
	release := make(chan struct{})
	if err := s.Add(Job{ID: "job", Run: func(context.Context) error {
		close(started)
		<-release
		return errors.New("retry")
	}}); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		s.runDue(context.Background(), time.Now())
		close(done)
	}()
	<-started
	s.Cancel("job")
	close(release)
	<-done
	s.mu.Lock()
	_, exists := s.jobs["job"]
	s.mu.Unlock()
	if exists {
		t.Fatal("cancelled job was reinserted after failure")
	}
}

func TestDispatcherCancellationInterruptsErrorBackoff(t *testing.T) {
	d := NewDispatcher()
	called := make(chan struct{})
	d.Register("recover", func(context.Context, string) error {
		close(called)
		return errors.New("publisher unavailable")
	})
	d.Enqueue("recover", "lot-1")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()
	<-called
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("dispatcher error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("dispatcher ignored cancellation during retry delay")
	}
}

func TestDispatcherHealthIsSafeUnderConcurrentReads(t *testing.T) {
	d := NewDispatcher()
	ctx, cancel := context.WithCancel(context.Background())
	d.Register("ok", func(context.Context, string) error { return nil })
	for i := 0; i < 20; i++ {
		d.Enqueue("ok", "lot")
	}
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = d.Health()
			}
		}()
	}
	wg.Wait()
	cancel()
	<-done
}
