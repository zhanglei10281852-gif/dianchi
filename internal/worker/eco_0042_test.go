package worker

import (
	"context"
	"testing"
	"time"
)

func TestEcoSchedulerKeepsReplacementJob(t *testing.T) {
	s := NewScheduler()
	now := time.Now()
	ran := 0
	s.jobs["j"] = Job{ID: "j", Next: now.Add(-time.Second), Run: func(context.Context) error {
		ran++
		s.mu.Lock()
		s.jobs["j"] = Job{ID: "j", Next: now.Add(time.Hour), Run: func(context.Context) error { return nil }}
		s.mu.Unlock()
		return nil
	}}
	s.runDue(context.Background(), now)
	if ran != 1 {
		t.Fatal(ran)
	}
	s.mu.Lock()
	_, ok := s.jobs["j"]
	s.mu.Unlock()
	if !ok {
		t.Fatal("replacement deleted")
	}
}
