package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/pagination"
	"github.com/zhanglei10281852-gif/dianchi/internal/storage/sqlite"
)

func TestCanceledLotListDoesNotReadPersistedRows(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := New(store, clock.Fixed{Value: time.Now()})
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	if _, err := svc.Intake(ctx, p, "LOT-26", "LFP", 5, nil, "request-26"); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	page, err := svc.List(canceled, p, pagination.Query{Limit: 10})
	if err == nil || len(page.Items) != 0 {
		t.Fatalf("canceled list returned rows: page=%+v err=%v", page, err)
	}
	live, err := svc.List(ctx, p, pagination.Query{Limit: 10})
	if err != nil || live.Total != 1 {
		t.Fatalf("live list failed: page=%+v err=%v", live, err)
	}
}
