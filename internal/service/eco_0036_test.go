package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/dispatch"
	"testing"
	"time"
)

func TestEcoRouteAdvancePropagatesStoreFailure(t *testing.T) {
	st := newReconciliationMemory()
	p, _ := dispatch.NewPlanner([]dispatch.Vehicle{{ID: "v", CapacityKg: 100, HazardLimit: 8, AvailableFrom: recNow()}})
	s := NewRouteService(p, st, recNow)
	op, _ := recPrincipals()
	r, _ := s.Plan(context.Background(), op, "eco-0036", "v")
	stop := dispatch.Stop{ID: "s", LotID: "l", Address: "yard", Hazard: 2, WeightKg: 10, WindowStart: recNow(), WindowEnd: recNow().Add(time.Hour)}
	if _, e := s.AddStop(context.Background(), op, r.ID, stop); e != nil {
		t.Fatal(e)
	}
	st.fail = true
	if _, e := s.Advance(context.Background(), op, r.ID, dispatch.Loading); e == nil {
		t.Fatal("swallowed")
	}
}
