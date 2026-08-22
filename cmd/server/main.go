package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/config"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/httpapi"
	"github.com/zhanglei10281852-gif/dianchi/internal/service"
	"github.com/zhanglei10281852-gif/dianchi/internal/storage/sqlite"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	store, err := sqlite.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	ttl, _ := time.ParseDuration(cfg.SessionTTL)
	authn := service.Auth{Store: store, Clock: clock.Real{}, SessionTTL: ttl}
	svc := service.New(store, clock.Real{})
	seed(ctx, authn)
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: httpapi.New(svc, authn).Handler()}
	go func() {
		log.Printf("dianchi listening on %s", cfg.Port)
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatal(e)
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
func seed(ctx context.Context, a service.Auth) {
	_ = a.Register(ctx, auth.Principal{ID: "operator-demo", TenantID: "demo", Name: "operator", Role: auth.Operator}, "operator")
	_ = a.Register(ctx, auth.Principal{ID: "supervisor-demo", TenantID: "demo", Name: "supervisor", Role: auth.Supervisor}, "supervisor")
}
