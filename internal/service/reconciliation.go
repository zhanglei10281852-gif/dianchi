package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/dispatch"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/reconciliation"
)

type ReconciliationStore interface {
	SaveReport(context.Context, reconciliation.Report) error
	LoadReport(context.Context, string, string) (reconciliation.Report, error)
	SaveRoute(context.Context, dispatch.Route) error
	LoadRoute(context.Context, string, string) (dispatch.Route, error)
}

type ReconciliationService struct {
	Store  ReconciliationStore
	Clock  func() time.Time
	Policy reconciliation.Policy
	mu     sync.Mutex
	cache  map[string]reconciliation.Report
}

func NewReconciliationService(store ReconciliationStore, now func() time.Time) *ReconciliationService {
	if now == nil {
		now = time.Now
	}
	return &ReconciliationService{Store: store, Clock: now, Policy: reconciliation.DefaultPolicy(), cache: map[string]reconciliation.Report{}}
}

func (s *ReconciliationService) CreateReport(ctx context.Context, p auth.Principal, id string) (reconciliation.Report, error) {
	if !p.Can("recover") {
		return reconciliation.Report{}, errors.New("role cannot reconcile")
	}
	if err := ctx.Err(); err != nil {
		return reconciliation.Report{}, err
	}
	r, e := reconciliation.NewReport(id, p.TenantID, s.Clock())
	if e != nil {
		return reconciliation.Report{}, e
	}
	if s.Store != nil {
		if e = s.Store.SaveReport(ctx, r); e != nil {
			return reconciliation.Report{}, e
		}
	}
	s.mu.Lock()
	s.cache[id] = r
	s.mu.Unlock()
	return r, nil
}
func (s *ReconciliationService) AppendMeasurement(ctx context.Context, p auth.Principal, id string, m reconciliation.Measurement) (reconciliation.Report, error) {
	if !p.Can("recover") {
		return reconciliation.Report{}, errors.New("role cannot reconcile")
	}
	if err := ctx.Err(); err != nil {
		return reconciliation.Report{}, err
	}
	r, e := s.load(ctx, p, id)
	if e != nil {
		return reconciliation.Report{}, e
	}
	r, e = r.Append(m)
	if e != nil {
		return reconciliation.Report{}, e
	}
	return s.save(ctx, r)
}
func (s *ReconciliationService) Submit(ctx context.Context, p auth.Principal, id string) (reconciliation.Report, error) {
	if !p.Can("recover") {
		return reconciliation.Report{}, errors.New("role cannot reconcile")
	}
	r, e := s.load(ctx, p, id)
	if e != nil {
		return reconciliation.Report{}, e
	}
	r, e = reconciliation.EvaluateWithPolicy(s.Clock(), s.Policy, r)
	if e != nil {
		return reconciliation.Report{}, e
	}
	r, e = r.Submit(s.Clock())
	if e != nil {
		return reconciliation.Report{}, e
	}
	return s.save(ctx, r)
}
func (s *ReconciliationService) Review(ctx context.Context, p auth.Principal, id string, approve bool) (reconciliation.Report, error) {
	if !p.Can("certify") {
		return reconciliation.Report{}, errors.New("only supervisors review reconciliation")
	}
	if err := ctx.Err(); err != nil {
		return reconciliation.Report{}, err
	}
	r, e := s.load(ctx, p, id)
	if e != nil {
		return reconciliation.Report{}, e
	}
	if reconciliation.NeedsInvestigation(r) && r.State == reconciliation.Submitted {
		r, e = r.StartInvestigation()
		if e != nil {
			return reconciliation.Report{}, e
		}
		if !approve {
			return s.save(ctx, r)
		}
	}
	if approve {
		r, e = r.Approve(p.ID, s.Clock())
	} else {
		r, e = r.Reject(p.ID)
	}
	if e != nil {
		return reconciliation.Report{}, e
	}
	return s.save(ctx, r)
}
func (s *ReconciliationService) load(ctx context.Context, p auth.Principal, id string) (reconciliation.Report, error) {
	s.mu.Lock()
	r, ok := s.cache[id]
	s.mu.Unlock()
	if ok {
		if r.TenantID != p.TenantID {
			return reconciliation.Report{}, errors.New("report belongs to another tenant")
		}
		return r.Snapshot(), nil
	}
	if s.Store == nil {
		return reconciliation.Report{}, errors.New("report not found")
	}
	r, e := s.Store.LoadReport(ctx, id, p.TenantID)
	if e != nil {
		return reconciliation.Report{}, e
	}
	s.mu.Lock()
	s.cache[id] = r
	s.mu.Unlock()
	return r.Snapshot(), nil
}
func (s *ReconciliationService) save(ctx context.Context, r reconciliation.Report) (reconciliation.Report, error) {
	if err := ctx.Err(); err != nil {
		return reconciliation.Report{}, err
	}
	if s.Store != nil {
		if err := s.Store.SaveReport(ctx, r); err != nil {
			return reconciliation.Report{}, err
		}
	}
	s.mu.Lock()
	s.cache[r.ID] = r
	s.mu.Unlock()
	return r.Snapshot(), nil
}

type RouteService struct {
	Planner *dispatch.Planner
	Store   ReconciliationStore
	Clock   func() time.Time
}

func NewRouteService(planner *dispatch.Planner, store ReconciliationStore, now func() time.Time) *RouteService {
	if now == nil {
		now = time.Now
	}
	return &RouteService{Planner: planner, Store: store, Clock: now}
}
func (s *RouteService) Plan(ctx context.Context, p auth.Principal, id, vehicle string) (dispatch.Route, error) {
	if !p.Can("reserve") {
		return dispatch.Route{}, errors.New("role cannot plan transport")
	}
	if err := ctx.Err(); err != nil {
		return dispatch.Route{}, err
	}
	r, e := s.Planner.Create(id, p.TenantID, vehicle, s.Clock())
	if e != nil {
		return dispatch.Route{}, e
	}
	if s.Store != nil {
		if e = s.Store.SaveRoute(ctx, r); e != nil {
			return dispatch.Route{}, e
		}
	}
	return r, nil
}
func (s *RouteService) AddStop(ctx context.Context, p auth.Principal, id string, stop dispatch.Stop) (dispatch.Route, error) {
	if !p.Can("reserve") {
		return dispatch.Route{}, errors.New("role cannot plan transport")
	}
	if err := ctx.Err(); err != nil {
		return dispatch.Route{}, err
	}
	r, e := s.Planner.AddStop(id, stop)
	if e != nil {
		return dispatch.Route{}, e
	}
	if s.Store != nil {
		if e = s.Store.SaveRoute(ctx, r); e != nil {
			return dispatch.Route{}, e
		}
	}
	return r, nil
}
func (s *RouteService) Advance(ctx context.Context, p auth.Principal, id string, status dispatch.Status) (dispatch.Route, error) {
	if !p.Can("reserve") {
		return dispatch.Route{}, errors.New("role cannot dispatch transport")
	}
	if err := ctx.Err(); err != nil {
		return dispatch.Route{}, err
	}
	r, e := s.Planner.Transition(id, status, s.Clock())
	if e != nil {
		return dispatch.Route{}, fmt.Errorf("advance route: %w", e)
	}
	if s.Store != nil {
		if e = s.Store.SaveRoute(ctx, r); e != nil {
			return dispatch.Route{}, e
		}
	}
	return r, nil
}
