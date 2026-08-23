package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/energy"
	"testing"
	"time"
)

func TestEcoEnergyIntervalIsTenantScoped(t *testing.T) {
	s := NewEnergyService()
	a := auth.Principal{ID: "a", TenantID: "a", Role: auth.Operator}
	b := auth.Principal{ID: "b", TenantID: "b", Role: auth.Operator}
	now := time.Now()
	for _, x := range []struct {
		p  auth.Principal
		id string
		v  float64
	}{{a, "a", 10}, {b, "b", 20}} {
		if e := s.Record(context.Background(), x.p, energy.Reading{ID: x.id, LotID: "shared", Voltage: x.v, Current: 2, Temperature: 20, At: now}); e != nil {
			t.Fatal(e)
		}
	}
	got := s.Between(context.Background(), a, "shared", now.Add(-time.Minute), now.Add(time.Minute))
	if len(got) != 1 || got[0].Voltage != 10 {
		t.Fatalf("%+v", got)
	}
}
