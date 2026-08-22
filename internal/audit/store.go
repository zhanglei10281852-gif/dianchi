package audit

import (
	"context"
	"fmt"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/audit"
	"sync"
	"time"
)

type Store struct {
	mu       sync.RWMutex
	events   []audit.Event
	failNext error
}

func New() *Store                   { return &Store{events: make([]audit.Event, 0)} }
func (s *Store) FailNext(err error) { s.mu.Lock(); defer s.mu.Unlock(); s.failNext = err }
func (s *Store) Append(ctx context.Context, e audit.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return err
	}
	s.events = append(s.events, e)
	return nil
}
func (s *Store) List(ctx context.Context, tenant string) []audit.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]audit.Event, 0)
	for _, e := range s.events {
		if e.TenantID == tenant {
			out = append(out, e)
		}
	}
	return out
}
func (s *Store) Count(tenant string) int { return len(s.List(context.Background(), tenant)) }
func (s *Store) Export(tenant string) string {
	items := s.List(context.Background(), tenant)
	return fmt.Sprintf("tenant=%s events=%d generated=%s", tenant, len(items), time.Now().UTC().Format(time.RFC3339))
}
