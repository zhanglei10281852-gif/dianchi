package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/dispatch"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/reconciliation"
)

type reconciliationMemory struct {
	mu      sync.Mutex
	reports map[string]reconciliation.Report
	routes  map[string]dispatch.Route
	fail    bool
}

func newReconciliationMemory() *reconciliationMemory {
	return &reconciliationMemory{reports: map[string]reconciliation.Report{}, routes: map[string]dispatch.Route{}}
}
func (m *reconciliationMemory) SaveReport(ctx context.Context, r reconciliation.Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errors.New("store unavailable")
	}
	m.reports[r.ID] = r.Snapshot()
	return nil
}
func (m *reconciliationMemory) LoadReport(ctx context.Context, id, tenant string) (reconciliation.Report, error) {
	if err := ctx.Err(); err != nil {
		return reconciliation.Report{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reports[id]
	if !ok || r.TenantID != tenant {
		return reconciliation.Report{}, errors.New("report not found")
	}
	return r.Snapshot(), nil
}
func (m *reconciliationMemory) SaveRoute(ctx context.Context, r dispatch.Route) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errors.New("store unavailable")
	}
	m.routes[r.ID] = r.Snapshot()
	return nil
}
func (m *reconciliationMemory) LoadRoute(ctx context.Context, id, tenant string) (dispatch.Route, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.routes[id]
	if !ok || r.TenantID != tenant {
		return dispatch.Route{}, errors.New("route not found")
	}
	return r.Snapshot(), nil
}
func recPrincipals() (auth.Principal, auth.Principal) {
	return auth.Principal{ID: "op", TenantID: "t", Role: auth.Operator}, auth.Principal{ID: "sup", TenantID: "t", Role: auth.Supervisor}
}
func recNow() time.Time { return time.Date(2026, 1, 3, 4, 5, 6, 0, time.UTC) }
func recMeasurement(lot string, actual int) reconciliation.Measurement {
	return reconciliation.Measurement{LotID: lot, Material: reconciliation.Lithium, Expected: 100, Actual: actual, Unit: "g", MeasuredAt: recNow(), Source: "scale"}
}

func TestReconciliationServicePersistsAndReviews(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	op, sup := recPrincipals()
	ctx := context.Background()
	r, err := svc.CreateReport(ctx, op, "report-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AppendMeasurement(ctx, op, r.ID, recMeasurement("lot-1", 99)); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Submit(ctx, op, r.ID); err != nil {
		t.Fatal(err)
	}
	out, err := svc.Review(ctx, sup, r.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if out.State != reconciliation.Approved {
		t.Fatalf("state=%s", out.State)
	}
	if store.reports[r.ID].State != reconciliation.Approved {
		t.Fatal("not persisted")
	}
}
func TestReconciliationDiscrepancyRequiresInvestigation(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	op, sup := recPrincipals()
	ctx := context.Background()
	r, _ := svc.CreateReport(ctx, op, "report-2")
	_, _ = svc.AppendMeasurement(ctx, op, r.ID, recMeasurement("lot-1", 10))
	if _, err := svc.Submit(ctx, op, r.ID); err != nil {
		t.Fatal(err)
	}
	out, err := svc.Review(ctx, sup, r.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.State != reconciliation.Investigating {
		t.Fatalf("state=%s", out.State)
	}
	if _, err := svc.Review(ctx, sup, r.ID, true); err == nil {
		t.Fatal("dirty report approved")
	}
}
func TestReconciliationRejectsTenantAndCancellation(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	op, _ := recPrincipals()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.CreateReport(ctx, op, "x"); err == nil {
		t.Fatal("cancelled create succeeded")
	}
	r, _ := svc.CreateReport(context.Background(), op, "x2")
	other := op
	other.TenantID = "other"
	if _, err := svc.AppendMeasurement(context.Background(), other, r.ID, recMeasurement("l", 1)); err == nil {
		t.Fatal("tenant crossed")
	}
}
func TestReconciliationStoreErrorsReachCaller(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	op, _ := recPrincipals()
	store.fail = true
	if _, err := svc.CreateReport(context.Background(), op, "x"); err == nil {
		t.Fatal("store error swallowed")
	}
}
func TestRouteServicePlansAndAdvances(t *testing.T) {
	store := newReconciliationMemory()
	v := dispatch.Vehicle{ID: "truck", CapacityKg: 100, HazardLimit: 7, AvailableFrom: recNow()}
	p, err := dispatch.NewPlanner([]dispatch.Vehicle{v})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewRouteService(p, store, recNow)
	op, _ := recPrincipals()
	ctx := context.Background()
	r, err := svc.Plan(ctx, op, "route-1", "truck")
	if err != nil {
		t.Fatal(err)
	}
	stop := dispatch.Stop{ID: "stop-1", LotID: "lot-1", Address: "A", Hazard: 3, WeightKg: 20, WindowStart: recNow(), WindowEnd: recNow().Add(time.Hour)}
	if _, err = svc.AddStop(ctx, op, r.ID, stop); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Advance(ctx, op, r.ID, dispatch.Loading); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Advance(ctx, op, r.ID, dispatch.EnRoute); err != nil {
		t.Fatal(err)
	}
	out, err := svc.Advance(ctx, op, r.ID, dispatch.Delivered)
	if err != nil || out.Status != dispatch.Delivered {
		t.Fatalf("out=%+v err=%v", out, err)
	}
}
func TestRouteServiceErrorsAndRace(t *testing.T) {
	v := dispatch.Vehicle{ID: "truck", CapacityKg: 100, HazardLimit: 7, AvailableFrom: recNow()}
	p, _ := dispatch.NewPlanner([]dispatch.Vehicle{v})
	svc := NewRouteService(p, nil, recNow)
	op, _ := recPrincipals()
	if _, err := svc.Plan(context.Background(), op, "r", "missing"); err == nil {
		t.Fatal("missing vehicle accepted")
	}
	r, _ := svc.Plan(context.Background(), op, "r", "truck")
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := svc.AddStop(context.Background(), op, r.ID, dispatch.Stop{ID: "same", LotID: "lot", Address: "A", Hazard: 1, WeightKg: 1, WindowStart: recNow(), WindowEnd: recNow().Add(time.Hour)})
			errs <- e
		}()
	}
	wg.Wait()
	close(errs)
	n := 0
	for e := range errs {
		if e == nil {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("winners=%d", n)
	}
}
