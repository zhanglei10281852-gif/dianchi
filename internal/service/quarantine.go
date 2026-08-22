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
	l.TenantID = p.TenantID
	q.mu.Lock()
	defer q.mu.Unlock()
	key := quarantineKey(p.TenantID, l.ID)
	q.items[key] = l
	q.notes[key] = append(q.notes[key], note)
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
	key := quarantineKey(p.TenantID, id)
	if _, ok := q.items[key]; !ok {
		return apperr.New(apperr.NotFound, "quarantine record missing")
	}
	delete(q.items, key)
	delete(q.notes, key)
	return nil
}
func (q *Quarantine) Get(p auth.Principal, id string) (battery.Lot, []string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	key := quarantineKey(p.TenantID, id)
	v, ok := q.items[key]
	if !ok {
		return battery.Lot{}, nil, false
	}
	return v, append([]string(nil), q.notes[key]...), true
}

func quarantineKey(tenant, lot string) string { return tenant + "\x00" + lot }
func (q *Quarantine) Expire(before time.Time) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := 0
	for id, l := range q.items {
		if l.ReceivedAt.Before(before) {
			delete(q.items, id)
			n++
		}
	}
	return n
}
