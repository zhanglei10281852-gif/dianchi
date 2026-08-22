package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"testing"
	"time"
)

func TestEcoAnalyticsHonorsCancellation(t *testing.T) {
	a := &Analytics{}
	p := auth.Principal{TenantID: "e"}
	if e := a.Record(context.Background(), p, battery.Received, time.Now()); e != nil {
		t.Fatal(e)
	}
	ctx, c := context.WithCancel(context.Background())
	c()
	if got := a.Daily(ctx, "e", time.Now().Add(-time.Hour), time.Now().Add(time.Hour)); got != nil {
		t.Fatalf("%+v", got)
	}
}
