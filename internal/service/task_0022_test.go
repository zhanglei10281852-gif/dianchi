package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/storage/sqlite"
)

func TestIntakeAuditFailureRollsBackLot(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.DB.ExecContext(ctx, "DROP TABLE audit_events"); err != nil {
		t.Fatal(err)
	}
	now := clock.Fixed{Value: time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)}
	svc := New(store, now)
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	if _, err := svc.Intake(ctx, p, "LOT-22", "LFP", 10, nil, "request-22"); err == nil {
		t.Fatal("intake succeeded without audit storage")
	}
	var count int
	if err := store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM lots WHERE code='LOT-22'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit failure committed %d lot row", count)
	}
}
