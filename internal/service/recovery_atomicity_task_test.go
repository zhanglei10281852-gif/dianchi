package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/audit"
	"github.com/zhanglei10281852-gif/dianchi/internal/repository"
)

type auditFailureStore struct {
	repository.Store
	fail bool
}

func (s *auditFailureStore) InsertAudit(ctx context.Context, tx *sql.Tx, e audit.Event) error {
	if s.fail {
		return errors.New("environmental audit sink unavailable")
	}
	return s.Store.InsertAudit(ctx, tx, e)
}

func (s *auditFailureStore) InsertAuditDirect(ctx context.Context, e audit.Event) error {
	if s.fail {
		return errors.New("environmental audit sink unavailable")
	}
	w, ok := s.Store.(interface {
		InsertAuditDirect(context.Context, audit.Event) error
	})
	if !ok {
		return errors.New("direct audit writer unavailable")
	}
	return w.InsertAuditDirect(ctx, e)
}

func TestRecoveryAuditFailurePreservesEcologicalLedger(t *testing.T) {
	f := newFixture(t)
	defer f.close()
	lot := createFlow(t, f)
	failing := &auditFailureStore{Store: f.store, fail: true}
	svc := New(failing, clock.Fixed{Value: f.svc.Clock.Now()})

	if _, err := svc.Recover(f.ctx, f.operator, lot.ID, 90, 55, 20, "eco-recovery-1", "eco-request-1"); err == nil {
		t.Fatal("recovery succeeded while the environmental audit sink was unavailable")
	}
	assertRecoveryLedger(t, f, lot.ID, "dismantling", 0, 0, 0, 0)

	failing.fail = false
	if _, err := svc.Recover(f.ctx, f.operator, lot.ID, 90, 55, 20, "eco-recovery-1", "eco-request-2"); err != nil {
		t.Fatalf("retry recovery: %v", err)
	}
	assertRecoveryLedger(t, f, lot.ID, "recovered", 1, 1, 1, 1)
}

func assertRecoveryLedger(t *testing.T, f fixture, lotID, wantState string, recoveries, outbox, idempotency, audits int) {
	t.Helper()
	var state string
	if err := f.store.DB.QueryRow(`SELECT state FROM lots WHERE id=?`, lotID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != wantState {
		t.Fatalf("lot state=%q, want %q", state, wantState)
	}
	checks := []struct {
		name  string
		query string
		want  int
	}{
		{"recoveries", `SELECT COUNT(*) FROM recoveries WHERE lot_id=?`, recoveries},
		{"outbox", `SELECT COUNT(*) FROM outbox WHERE aggregate_id=?`, outbox},
		{"idempotency", `SELECT COUNT(*) FROM idempotency_keys WHERE tenant_id=? AND key=? AND operation=?`, idempotency},
		{"recover audits", `SELECT COUNT(*) FROM audit_events WHERE object_id=? AND action='recover'`, audits},
	}
	for _, check := range checks {
		var got int
		var err error
		switch check.name {
		case "idempotency":
			err = f.store.DB.QueryRow(check.query, "tenant-a", "eco-recovery-1", "recover").Scan(&got)
		default:
			err = f.store.DB.QueryRow(check.query, lotID).Scan(&got)
		}
		if err != nil {
			t.Fatalf("%s query: %v", check.name, err)
		}
		if got != check.want {
			t.Fatalf("%s=%d, want %d", check.name, got, check.want)
		}
	}
}

var _ repository.Store = (*auditFailureStore)(nil)
var _ interface {
	InsertAuditDirect(context.Context, audit.Event) error
} = (*auditFailureStore)(nil)
