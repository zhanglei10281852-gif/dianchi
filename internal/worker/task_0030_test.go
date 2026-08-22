package worker

import (
	"context"
	"testing"
	"time"
)

func TestSchedulerCancellationReachesRunningJob(t *testing.T) {
	scheduler := NewScheduler()
	started := make(chan context.Context, 1)
	release := make(chan struct{})
	if err := scheduler.Add(Job{ID: "job-30", Run: func(ctx context.Context) error {
		started <- ctx
		<-release
		return ctx.Err()
	}}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx, time.Millisecond) }()
	jobCtx := <-started
	cancel()
	select {
	case <-jobCtx.Done():
	case <-time.After(100 * time.Millisecond):
		close(release)
		t.Fatal("scheduler cancellation did not reach running job")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop")
	}
}
