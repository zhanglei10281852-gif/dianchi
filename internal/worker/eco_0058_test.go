package worker

import (
	"context"
	"testing"
	"time"
)

func TestEcoSchedulerStopsCanceledJob(t *testing.T) {
	s := NewScheduler()
	now := time.Now()
	ctx, c := context.WithCancel(context.Background())
	c()
	ran := false
	s.jobs["j"] = Job{ID: "j", Next: now.Add(-time.Second), Run: func(x context.Context) error {
		ran = true
		if x.Err() == nil {
			t.Fatal("lost cancel")
		}
		return x.Err()
	}}
	s.runDue(ctx, now)
	if ran && s.jobs["j"].Attempts == 0 {
		t.Fatal("executed canceled job")
	}
}
