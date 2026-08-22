package worker

import (
	"context"
	"sync"
	"time"
)

type Job struct {
	ID       string
	Run      func(context.Context) error
	Attempts int
	Next     time.Time
}
type Scheduler struct {
	mu   sync.Mutex
	jobs map[string]Job
	done chan struct{}
}

func NewScheduler() *Scheduler { return &Scheduler{jobs: map[string]Job{}, done: make(chan struct{})} }
func (s *Scheduler) Add(j Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[j.ID]; ok {
		return ErrDuplicate
	}
	s.jobs[j.ID] = j
	return nil
}
func (s *Scheduler) Cancel(id string) { s.mu.Lock(); defer s.mu.Unlock(); delete(s.jobs, id) }
func (s *Scheduler) Run(ctx context.Context, interval time.Duration) error {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.done:
			return nil
		case now := <-tick.C:
			s.runDue(ctx, now)
		}
	}
}
func (s *Scheduler) runDue(ctx context.Context, now time.Time) {
	s.mu.Lock()
	jobs := make([]Job, 0)
	for _, j := range s.jobs {
		if j.Next.IsZero() || !j.Next.After(now) {
			jobs = append(jobs, j)
		}
	}
	s.mu.Unlock()
	for _, j := range jobs {
		err := j.Run(ctx)
		s.mu.Lock()
		current, stillScheduled := s.jobs[j.ID]
		if !stillScheduled || current.Attempts != j.Attempts || !current.Next.Equal(j.Next) {
			s.mu.Unlock()
			continue
		}
		if err == nil {
			delete(s.jobs, j.ID)
		} else {
			j.Attempts++
			j.Next = now.Add(time.Duration(j.Attempts) * time.Second)
			s.jobs[j.ID] = j
		}
		s.mu.Unlock()
	}
}
func (s *Scheduler) Stop() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

type workerError string

func (e workerError) Error() string { return string(e) }

var ErrDuplicate = workerError("duplicate job")
