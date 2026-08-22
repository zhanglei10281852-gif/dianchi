package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"github.com/zhanglei10281852-gif/dianchi/internal/pagination"
	"github.com/zhanglei10281852-gif/dianchi/internal/storage/sqlite"
)

type fixture struct {
	ctx                  context.Context
	store                *sqlite.Store
	svc                  *Service
	auth                 Auth
	operator, supervisor auth.Principal
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	st, err := sqlite.Open(ctx, fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	c := clock.Fixed{Value: time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)}
	a := Auth{Store: st, Clock: c, SessionTTL: time.Hour}
	op := auth.Principal{ID: "op-1", TenantID: "tenant-a", Name: "op", Role: auth.Operator}
	sp := auth.Principal{ID: "sup-1", TenantID: "tenant-a", Name: "sup", Role: auth.Supervisor}
	if err := a.Register(ctx, op, "op-secret"); err != nil {
		t.Fatal(err)
	}
	if err := a.Register(ctx, sp, "sup-secret"); err != nil {
		t.Fatal(err)
	}
	return fixture{ctx: ctx, store: st, svc: New(st, c), auth: a, operator: op, supervisor: sp}
}
func (f fixture) close() { _ = f.store.Close() }
func createFlow(t *testing.T, f fixture) battery.Lot {
	t.Helper()
	lot, err := f.svc.Intake(f.ctx, f.operator, "LOT-001", "NMC", 20, nil, "req-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.Inspect(f.ctx, f.operator, lot.ID, battery.InspectionPass, "visual pass", "req-2"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.Reserve(f.ctx, f.operator, lot.ID, "station-7", "req-3"); err != nil {
		t.Fatal(err)
	}
	if err = f.svc.BeginDismantling(f.ctx, f.operator, lot.ID, "req-4"); err != nil {
		t.Fatal(err)
	}
	return lot
}
func TestLifecycleReachesCertified(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot := createFlow(t, f)
	if _, err := f.svc.Recover(f.ctx, f.operator, lot.ID, 120, 80, 40, "key-1", "req-5"); err != nil {
		t.Fatal(err)
	}
	cert, err := f.svc.Certify(f.ctx, f.supervisor, lot.ID, "req-6")
	if err != nil {
		t.Fatal(err)
	}
	if cert.State != "issued" {
		t.Fatalf("state=%s", cert.State)
	}
	var state string
	if err = f.store.DB.QueryRow(`SELECT state FROM lots WHERE id=?`, lot.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "certified" {
		t.Fatalf("persisted state=%s", state)
	}
}
func TestInspectionHoldQuarantinesLot(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot, err := f.svc.Intake(f.ctx, f.operator, "LOT-HOLD", "LFP", 80, nil, "r")
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.svc.Inspect(f.ctx, f.operator, lot.ID, battery.InspectionHold, "swollen case", "r2")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != battery.Quarantined {
		t.Fatalf("state=%s", out.State)
	}
	if _, err = f.svc.Reserve(f.ctx, f.operator, lot.ID, "station", "r3"); err == nil {
		t.Fatal("quarantined lot reserved")
	}
}
func TestTenantIsolation(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	other := auth.Principal{ID: "op-2", TenantID: "tenant-b", Name: "other", Role: auth.Operator}
	if err := f.auth.Register(f.ctx, other, "x"); err != nil {
		t.Fatal(err)
	}
	lot, err := f.svc.Intake(f.ctx, f.operator, "LOT-TENANT", "LFP", 1, nil, "r")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.Inspect(f.ctx, other, lot.ID, battery.InspectionPass, "", "r2"); err == nil {
		t.Fatal("cross tenant access")
	}
	if _, err = f.svc.List(f.ctx, other, pagination.Query{}); err != nil {
		t.Fatal(err)
	}
}
func TestIdempotentRecovery(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot := createFlow(t, f)
	a, err := f.svc.Recover(f.ctx, f.operator, lot.ID, 100, 50, 25, "same-key", "r")
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.svc.Recover(f.ctx, f.operator, lot.ID, 100, 50, 25, "same-key", "r2")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Fatalf("duplicate recovery %s %s", a.ID, b.ID)
	}
	var n int
	if err = f.store.DB.QueryRow(`SELECT COUNT(*) FROM recoveries WHERE lot_id=?`, lot.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("recoveries=%d", n)
	}
}
func TestExpiredSessionRejected(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	tok, _, err := f.auth.Login(f.ctx, "tenant-a", "op", "op-secret")
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.auth.Principal(f.ctx, tok)
	if err != nil || p.ID != "op-1" {
		t.Fatalf("principal=%v err=%v", p, err)
	}
	expired := Auth{Store: f.store, Clock: clock.Fixed{Value: time.Now().Add(2 * time.Hour)}, SessionTTL: time.Hour}
	if _, err = expired.Principal(f.ctx, tok); err == nil {
		t.Fatal("expired session accepted")
	}
}
func TestLogoutRevokesSession(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	tok, _, err := f.auth.Login(f.ctx, "tenant-a", "op", "op-secret")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.auth.Logout(f.ctx, tok); err != nil {
		t.Fatal(err)
	}
	if _, err = f.auth.Principal(f.ctx, tok); err == nil {
		t.Fatal("revoked session accepted")
	}
}
func TestInvalidStateTransitions(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot, err := f.svc.Intake(f.ctx, f.operator, "LOT-STATE", "LFP", 1, nil, "r")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.Reserve(f.ctx, f.operator, lot.ID, "s", "r2"); err == nil {
		t.Fatal("reserved before inspection")
	}
	var e *apperr.Error
	if !errors.As(err, &e) || e.Code != apperr.Conflict {
		t.Fatalf("wrong error %v", err)
	}
}
func TestListPaginationAndFilters(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	for i := 0; i < 5; i++ {
		if _, err := f.svc.Intake(f.ctx, f.operator, fmt.Sprintf("LOT-%d", i), "LFP", i, nil, fmt.Sprintf("r-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	p, err := f.svc.List(f.ctx, f.operator, pagination.Query{Limit: 2, Offset: 1, Chemistry: "LFP"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 5 || len(p.Items) != 2 {
		t.Fatalf("page=%+v", p)
	}
}
func TestConcurrentReservationSingleWinner(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot, err := f.svc.Intake(f.ctx, f.operator, "LOT-CONC", "LFP", 1, nil, "r")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.Inspect(f.ctx, f.operator, lot.ID, battery.InspectionPass, "", "r2"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, e := f.svc.Reserve(f.ctx, f.operator, lot.ID, fmt.Sprintf("station-%d", i), fmt.Sprintf("r-%d", i))
			errs <- e
		}(i)
	}
	wg.Wait()
	close(errs)
	success := 0
	for e := range errs {
		if e == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("reservation winners=%d", success)
	}
}
func TestContextCancellationStopsIntake(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	_, err := f.svc.Intake(ctx, f.operator, "LOT-CANCEL", "LFP", 1, nil, "r")
	if err == nil {
		t.Fatal("cancelled intake succeeded")
	}
}
func TestRecoverRejectsNegativeMaterial(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot := createFlow(t, f)
	_, err := f.svc.Recover(f.ctx, f.operator, lot.ID, -1, 2, 3, "k", "r")
	if err == nil {
		t.Fatal("negative material accepted")
	}
}
func TestSupervisorOnlyCertification(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot := createFlow(t, f)
	if _, err := f.svc.Certify(f.ctx, f.operator, lot.ID, "r"); err == nil {
		t.Fatal("operator certified")
	}
}
func TestDuplicateCertificateRejected(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot := createFlow(t, f)
	if _, err := f.svc.Recover(f.ctx, f.operator, lot.ID, 1, 1, 1, "k", "r"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Certify(f.ctx, f.supervisor, lot.ID, "r2"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Certify(f.ctx, f.supervisor, lot.ID, "r3"); err == nil {
		t.Fatal("duplicate certificate")
	}
}
func TestAuditRowsFollowOperations(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot := createFlow(t, f)
	if _, err := f.svc.Recover(f.ctx, f.operator, lot.ID, 1, 2, 3, "k", "r"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.store.DB.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE object_id=?`, lot.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 5 {
		t.Fatalf("audit count=%d", n)
	}
}
func TestRecoveryOutboxCreatedAtomically(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot := createFlow(t, f)
	if _, err := f.svc.Recover(f.ctx, f.operator, lot.ID, 2, 3, 4, "k", "r"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.store.DB.QueryRow(`SELECT COUNT(*) FROM outbox WHERE aggregate_id=?`, lot.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("outbox=%d", n)
	}
}
func TestFindMissingLotReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	_, err := f.svc.Inspect(f.ctx, f.operator, "missing", battery.InspectionPass, "", "r")
	var e *apperr.Error
	if !errors.As(err, &e) || e.Code != apperr.NotFound {
		t.Fatalf("err=%v", err)
	}
}
func TestRoleCannotUseUnknownAction(t *testing.T) {
	p := auth.Principal{Role: auth.Operator}
	if p.Can("audit") {
		t.Fatal("operator can audit")
	}
	if p.Can("unknown") {
		t.Fatal("unknown action allowed")
	}
}
func TestDatabaseSurvivesReopen(t *testing.T) {
	path := fmt.Sprintf("file:%s/persist.db", t.TempDir())
	ctx := context.Background()
	s, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	a := Auth{Store: s, Clock: clock.Real{}, SessionTTL: time.Hour}
	p := auth.Principal{ID: "persist", TenantID: "t", Name: "persist", Role: auth.Operator}
	if err = a.Register(ctx, p, "pw"); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	s2, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if _, _, err = a.Login(ctx, "t", "persist", "pw"); err == nil {
		t.Fatal("old auth service used closed store")
	}
	a.Store = s2
	if _, _, err = a.Login(ctx, "t", "persist", "pw"); err != nil {
		t.Fatal(err)
	}
}

var _ *sql.DB
