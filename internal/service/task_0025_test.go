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

func TestCanceledLoginDoesNotCreateSession(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	authService := Auth{Store: store, Clock: clock.Fixed{Value: time.Now()}, SessionTTL: time.Hour}
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Name: "op", Role: auth.Operator}
	if err := authService.Register(ctx, p, "secret"); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := authService.Login(canceled, p.TenantID, p.Name, "secret"); err == nil {
		t.Fatal("canceled login succeeded")
	}
	var count int
	if err := store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("canceled login created %d session", count)
	}
	if _, _, err := authService.Login(ctx, p.TenantID, p.Name, "secret"); err != nil {
		t.Fatalf("live login failed: %v", err)
	}
}
