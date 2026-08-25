package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/service"
	"github.com/zhanglei10281852-gif/dianchi/internal/storage/sqlite"
)

func TestCanceledHTTPIntakeDoesNotCreateLot(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	clk := clock.Fixed{Value: time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)}
	authService := service.Auth{Store: store, Clock: clk, SessionTTL: time.Hour}
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Name: "op", Role: auth.Operator}
	if err := authService.Register(ctx, p, "secret"); err != nil {
		t.Fatal(err)
	}
	token, _, err := authService.Login(ctx, p.TenantID, p.Name, "secret")
	if err != nil {
		t.Fatal(err)
	}
	handler := New(service.New(store, clk), authService).Handler()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/v1/lots", bytes.NewBufferString(`{"Code":"LOT-23","Chemistry":"LFP","Hazard":5}`)).WithContext(canceled)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	var count int
	if err := store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM lots WHERE code='LOT-23'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("canceled HTTP request created %d lot", count)
	}
}
