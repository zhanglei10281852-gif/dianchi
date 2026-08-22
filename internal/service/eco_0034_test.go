package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/dispatch"
	"testing"
)

func TestEcoRoutePlanPropagatesStoreFailure(t *testing.T) {
	st := newReconciliationMemory()
	st.fail = true
	p, _ := dispatch.NewPlanner([]dispatch.Vehicle{{ID: "v", CapacityKg: 100, HazardLimit: 8, AvailableFrom: recNow()}})
	s := NewRouteService(p, st, recNow)
	op, _ := recPrincipals()
	if _, e := s.Plan(context.Background(), op, "eco-0034", "v"); e == nil {
		t.Fatal("swallowed")
	}
}
