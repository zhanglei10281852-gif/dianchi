package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/energy"
	"sync"
	"time"
)

type EnergyService struct {
	mu       sync.RWMutex
	readings map[string][]energy.Reading
}

func NewEnergyService() *EnergyService {
	return &EnergyService{readings: map[string][]energy.Reading{}}
}
func (e *EnergyService) Record(ctx context.Context, p auth.Principal, r energy.Reading) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("inspect") {
		return apperr.New(apperr.Forbidden, "role cannot record energy")
	}
	if err := energy.Validate(r); err != nil {
		return apperr.Wrap(apperr.Invalid, "reading", err)
	}
	r.TenantID = p.TenantID
	e.mu.Lock()
	defer e.mu.Unlock()
	e.readings[r.LotID] = append(e.readings[r.LotID], r)
	return nil
}
func (e *EnergyService) Average(ctx context.Context, p auth.Principal, lot string) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	rows := e.readings[lot]
	if len(rows) == 0 {
		return 0, apperr.New(apperr.NotFound, "readings missing")
	}
	out, err := energy.Aggregate(rows)
	return out, err
}
func (e *EnergyService) Between(ctx context.Context, lot string, start, end time.Time) []energy.Reading {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]energy.Reading, 0)
	for _, r := range e.readings[lot] {
		if !r.At.Before(start) && r.At.Before(end) {
			out = append(out, r)
		}
	}
	return out
}
