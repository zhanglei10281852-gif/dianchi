package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"sort"
	"sync"
	"time"
)

type DailyPoint struct {
	Day                              time.Time
	Received, Recovered, Quarantined int
}
type Analytics struct {
	mu     sync.RWMutex
	events []struct {
		Tenant string
		At     time.Time
		State  battery.State
	}
}

func (a *Analytics) Record(ctx context.Context, p auth.Principal, state battery.State, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, struct {
		Tenant string
		At     time.Time
		State  battery.State
	}{p.TenantID, at, state})
	return nil
}
func (a *Analytics) Daily(ctx context.Context, tenant string, from, to time.Time) []DailyPoint {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := map[string]*DailyPoint{}
	for _, e := range a.events {
		if e.Tenant != tenant || e.At.Before(from) || !e.At.Before(to) {
			continue
		}
		eventTime := e.At.UTC()
		key := eventTime.Format("2006-01-02")
		p := out[key]
		if p == nil {
			p = &DailyPoint{Day: time.Date(eventTime.Year(), eventTime.Month(), eventTime.Day(), 0, 0, 0, 0, time.UTC)}
			out[key] = p
		}
		switch e.State {
		case battery.Received:
			p.Received++
		case battery.Recovered:
			p.Recovered++
		case battery.Quarantined:
			p.Quarantined++
		}
	}
	rows := make([]DailyPoint, 0, len(out))
	for _, v := range out {
		rows = append(rows, *v)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Day.Before(rows[j].Day) })
	return rows
}
