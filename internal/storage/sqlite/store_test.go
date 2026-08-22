package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/audit"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"github.com/zhanglei10281852-gif/dianchi/internal/pagination"
)

func storeFixture(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestMigrationsCreateRelationalTables(t *testing.T) {
	s := storeFixture(t)
	defer s.Close()
	rows, err := s.DB.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		seen[n] = true
	}
	for _, n := range []string{"operators", "sessions", "lots", "inspections", "reservations", "recoveries", "certificates", "audit_events", "idempotency_keys", "outbox"} {
		if !seen[n] {
			t.Fatalf("missing table %s", n)
		}
	}
}
func TestStoreRoundTripLot(t *testing.T) {
	s := storeFixture(t)
	defer s.Close()
	ctx := context.Background()
	l := battery.Lot{ID: "l1", TenantID: "t", Code: "c", Chemistry: "LFP", State: battery.Received, Version: 1, ReceivedAt: time.Now().UTC(), CreatedBy: "u"}
	if err := s.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error { return s.InsertLot(ctx, tx, l) }); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetLot(ctx, nil, "l1", "t")
	if err != nil || got.Code != "c" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err = s.GetLot(ctx, nil, "l1", "other"); err == nil {
		t.Fatal("tenant leak")
	}
}
func TestListCountUsesFilters(t *testing.T) {
	s := storeFixture(t)
	defer s.Close()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		l := battery.Lot{ID: fmt.Sprintf("l%d", i), TenantID: "t", Code: fmt.Sprintf("c%d", i), Chemistry: "LFP", State: battery.Received, Version: 1, ReceivedAt: time.Now().UTC(), CreatedBy: "u"}
		if i == 2 {
			l.Chemistry = "NMC"
		}
		if err := s.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error { return s.InsertLot(ctx, tx, l) }); err != nil {
			t.Fatal(err)
		}
	}
	out, err := s.ListLots(ctx, "t", pagination.Query{Chemistry: "LFP", Limit: 10})
	if err != nil || out.Total != 2 || len(out.Items) != 2 {
		t.Fatalf("%+v %v", out, err)
	}
}
func TestAuditAndOutboxWrites(t *testing.T) {
	s := storeFixture(t)
	defer s.Close()
	ctx := context.Background()
	if err := s.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := s.InsertAudit(ctx, tx, audit.Event{ID: "a", TenantID: "t", ActorID: "u", ObjectType: "lot", ObjectID: "l", Action: "x", Result: "ok", RequestID: "r", Payload: "{}", CreatedAt: time.Now().UTC()}); err != nil {
			return err
		}
		return s.InsertOutbox(ctx, tx, "o", "t", "l", "event", "{}")
	}); err != nil {
		t.Fatal(err)
	}
	id, agg, p, err := s.ClaimOutbox(ctx, time.Now().UTC().Add(time.Second))
	if err != nil || id != "o" || agg != "l" || p != "{}" {
		t.Fatalf("%s %s %s %v", id, agg, p, err)
	}
	if err = s.MarkOutbox(ctx, id, nil); err != nil {
		t.Fatal(err)
	}
}
func TestSessionLifecycle(t *testing.T) {
	s := storeFixture(t)
	defer s.Close()
	ctx := context.Background()
	p := auth.Principal{ID: "u", TenantID: "t", Name: "n", Role: auth.Operator}
	if err := s.CreateOperator(ctx, p, "pw"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, "s", p, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := s.FindSession(ctx, "s", time.Now())
	if err != nil || got.ID != "u" {
		t.Fatal(err)
	}
	if err = s.RevokeSession(ctx, "s", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.FindSession(ctx, "s", time.Now()); err == nil {
		t.Fatal("revoked session")
	}
}
