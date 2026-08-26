package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/reconciliation"
)

func TestReconciliationPermissions(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	guest := auth.Principal{ID: "guest", TenantID: "t", Role: auth.Role("guest")}
	if _, err := svc.CreateReport(context.Background(), guest, "report"); err == nil {
		t.Fatal("guest created report")
	}
	op, sup := recPrincipals()
	r, err := svc.CreateReport(context.Background(), op, "report")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(context.Background(), op, r.ID, true); err == nil {
		t.Fatal("operator reviewed report")
	}
	if _, err := svc.AppendMeasurement(context.Background(), sup, r.ID, recMeasurement("lot", 99)); err != nil {
		t.Fatal(err)
	}
}

func TestReconciliationSaveChecksContextAfterDomainWork(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	op, _ := recPrincipals()
	r, err := svc.CreateReport(context.Background(), op, "cancel-save")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.AppendMeasurement(ctx, op, r.ID, recMeasurement("lot", 99)); err == nil {
		t.Fatal("cancelled append succeeded")
	}
}

func TestReconciliationReloadsPersistedState(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	op, _ := recPrincipals()
	r, err := svc.CreateReport(context.Background(), op, "reload")
	if err != nil {
		t.Fatal(err)
	}
	stored := store.reports[r.ID]
	stored, err = stored.Append(recMeasurement("lot", 99))
	if err != nil {
		t.Fatal(err)
	}
	store.reports[r.ID] = stored
	svc.mu.Lock()
	delete(svc.cache, r.ID)
	svc.mu.Unlock()
	got, err := svc.AppendMeasurement(context.Background(), op, r.ID, recMeasurement("lot-2", 98))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Measurements) != 2 {
		t.Fatalf("reloaded measurements=%d", len(got.Measurements))
	}
}

func TestReconciliationPolicyCanBeCustomized(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	svc.Policy.ToleranceByMaterial[reconciliation.Lithium] = .2
	op, sup := recPrincipals()
	r, _ := svc.CreateReport(context.Background(), op, "custom-policy")
	if _, err := svc.AppendMeasurement(context.Background(), op, r.ID, recMeasurement("lot", 85)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(context.Background(), op, r.ID); err != nil {
		t.Fatal(err)
	}
	out, err := svc.Review(context.Background(), sup, r.ID, true)
	if err != nil || out.State != reconciliation.Approved {
		t.Fatalf("custom policy result=%+v err=%v", out, err)
	}
}

func TestReconciliationReportVersionIncrementsAcrossWorkflow(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	op, sup := recPrincipals()
	r, _ := svc.CreateReport(context.Background(), op, "versions")
	if r.Version != 1 {
		t.Fatalf("initial version=%d", r.Version)
	}
	r, _ = svc.AppendMeasurement(context.Background(), op, r.ID, recMeasurement("lot", 99))
	if r.Version != 1 {
		t.Fatalf("append changed version=%d", r.Version)
	}
	r, _ = svc.Submit(context.Background(), op, r.ID)
	if r.Version != 2 {
		t.Fatalf("submit version=%d", r.Version)
	}
	r, _ = svc.Review(context.Background(), sup, r.ID, true)
	if r.Version != 3 {
		t.Fatalf("approval version=%d", r.Version)
	}
}

func TestReconciliationReturnedReportDoesNotExposeCache(t *testing.T) {
	store := newReconciliationMemory()
	svc := NewReconciliationService(store, recNow)
	op, _ := recPrincipals()
	r, _ := svc.CreateReport(context.Background(), op, "cache-copy")
	r, err := svc.AppendMeasurement(context.Background(), op, r.ID, recMeasurement("lot", 99))
	if err != nil {
		t.Fatal(err)
	}
	r.Measurements[0].Actual = 1
	loaded, err := svc.load(context.Background(), op, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Measurements[0].Actual != 99 {
		t.Fatalf("caller polluted cached measurement: %+v", loaded.Measurements[0])
	}
}
