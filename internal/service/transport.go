package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/transport"
	"sync"
	"time"
)

type TransportService struct {
	mu       sync.RWMutex
	trips    map[string]transport.Trip
	vehicles map[string]transport.Vehicle
}

func NewTransportService() *TransportService {
	return &TransportService{trips: map[string]transport.Trip{}, vehicles: map[string]transport.Vehicle{}}
}
func (t *TransportService) AddVehicle(ctx context.Context, p auth.Principal, v transport.Vehicle) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("reserve") {
		return apperr.New(apperr.Forbidden, "role cannot add vehicle")
	}
	if err := transport.ValidateVehicle(v); err != nil {
		return apperr.Wrap(apperr.Invalid, "vehicle", err)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.vehicles[v.ID] = v
	return nil
}
func (t *TransportService) Plan(ctx context.Context, p auth.Principal, v transport.Trip) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("reserve") {
		return apperr.New(apperr.Forbidden, "role cannot plan trip")
	}
	v.TenantID = p.TenantID
	v.Status = transport.Planned
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.trips[v.ID]; ok {
		return apperr.New(apperr.Conflict, "trip exists")
	}
	t.trips[v.ID] = v
	return nil
}
func (t *TransportService) AddStop(ctx context.Context, p auth.Principal, id string, s transport.Stop) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := transport.ValidateStop(s); err != nil {
		return apperr.Wrap(apperr.Invalid, "stop", err)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	v, ok := t.trips[id]
	if !ok || v.TenantID != p.TenantID {
		return apperr.New(apperr.NotFound, "trip missing")
	}
	if v.Status != transport.Planned {
		return apperr.New(apperr.Conflict, "trip already loading")
	}
	s.TripID = id
	v.Stops = append(v.Stops, s)
	t.trips[id] = v
	return nil
}
func (t *TransportService) Move(ctx context.Context, p auth.Principal, id string, target transport.Status) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	v, ok := t.trips[id]
	if !ok || v.TenantID != p.TenantID {
		return apperr.New(apperr.NotFound, "trip missing")
	}
	if err := transport.Transition(v.Status, target); err != nil {
		return apperr.Wrap(apperr.Conflict, "trip transition", err)
	}
	v.Status = target
	now := time.Now().UTC()
	if target == transport.InTransit {
		v.DepartedAt = &now
	}
	if target == transport.Arrived {
		v.ArrivedAt = &now
	}
	t.trips[id] = v
	return nil
}
func (t *TransportService) Get(ctx context.Context, p auth.Principal, id string) (transport.Trip, error) {
	if err := ctx.Err(); err != nil {
		return transport.Trip{}, err
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	v, ok := t.trips[id]
	if !ok || v.TenantID != p.TenantID {
		return transport.Trip{}, apperr.New(apperr.NotFound, "trip missing")
	}
	v.Stops = append([]transport.Stop(nil), v.Stops...)
	return v, nil
}
