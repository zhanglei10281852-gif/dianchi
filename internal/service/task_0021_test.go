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

type task21Store struct {
	mu      sync.Mutex
	reports map[string]reconciliation.Report
	fail    bool
}

func (s *task21Store) SaveReport(ctx context.Context, report reconciliation.Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return errors.New("persistence unavailable")
	}
	s.reports[report.ID] = report.Snapshot()
	return nil
}
func (s *task21Store) LoadReport(ctx context.Context, id, tenant string) (reconciliation.Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report, ok := s.reports[id]
	if !ok || report.TenantID != tenant {
		return reconciliation.Report{}, errors.New("report missing")
	}
	return report.Snapshot(), ctx.Err()
}
func (s *task21Store) SaveRoute(context.Context, dispatch.Route) error { return nil }
func (s *task21Store) LoadRoute(context.Context, string, string) (dispatch.Route, error) {
	return dispatch.Route{}, errors.New("route missing")
}

func TestFailedReconciliationSaveDoesNotPolluteCache(t *testing.T) {
	ctx := context.Background()
	store := &task21Store{reports: map[string]reconciliation.Report{}}
	now := func() time.Time { return time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC) }
	svc := NewReconciliationService(store, now)
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	report, err := svc.CreateReport(ctx, p, "report-21")
	if err != nil {
		t.Fatal(err)
	}
	measurement := reconciliation.Measurement{LotID: "lot-21", Material: reconciliation.Lithium, Expected: 100, Actual: 99, Unit: "g", MeasuredAt: now(), Source: "scale"}
	store.fail = true
	if _, err := svc.AppendMeasurement(ctx, p, report.ID, measurement); err == nil {
		t.Fatal("append succeeded while persistence failed")
	}
	store.fail = false
	got, err := svc.AppendMeasurement(ctx, p, report.ID, measurement)
	if err != nil || len(got.Measurements) != 1 {
		t.Fatalf("failed save polluted retry state: report=%+v err=%v", got, err)
	}
}
