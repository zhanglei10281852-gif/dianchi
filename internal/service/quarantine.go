package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"sync"
	"time"
)

type Quarantine struct {
	mu    sync.Mutex
	items map[string]battery.Lot
	notes map[string][]string
}

func NewQuarantine() *Quarantine {
	return &Quarantine{items: map[string]battery.Lot{}, notes: map[string][]string{}}
}
func (q *Quarantine) Hold(ctx context.Context, p auth.Principal, l battery.Lot, note string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("inspect") {
		return apperr.New(apperr.Forbidden, "role cannot quarantine")
	}
	if l.State != battery.Quarantined {
		return apperr.New(apperr.Invalid, "lot is not quarantined")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items[l.ID] = l
	q.notes[l.ID] = append(q.notes[l.ID], note)
	return nil
}
func (q *Quarantine) Release(ctx context.Context, p auth.Principal, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("certify") {
		return apperr.New(apperr.Forbidden, "only supervisors release")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.items[id]; !ok {
		return apperr.New(apperr.NotFound, "quarantine record missing")
	}
	delete(q.items, id)
	return nil
}
func (q *Quarantine) Get(id string) (battery.Lot, []string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	v, ok := q.items[id]
	if !ok {
		return battery.Lot{}, nil, false
	}
	return v, append([]string(nil), q.notes[id]...), true
}
func (q *Quarantine) Expire(before time.Time) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := 0
	for id, l := range q.items {
		if l.ReceivedAt.Before(before) {
			delete(q.items, id)
			delete(q.notes, id)
			n++
		}
	}
	return n
}
