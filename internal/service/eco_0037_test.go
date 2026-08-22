package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/dispatch"
	"testing"
)

func TestEcoRoutePlanKeepsTenantIdentity(t *testing.T) {
	p, _ := dispatch.NewPlanner([]dispatch.Vehicle{{ID: "v", CapacityKg: 100, HazardLimit: 8, AvailableFrom: recNow()}})
	s := NewRouteService(p, nil, recNow)
	op, _ := recPrincipals()
	got, e := s.Plan(context.Background(), op, "eco-0037", "v")
	if e != nil {
		t.Fatal(e)
	}
	if got.TenantID != op.TenantID {
		t.Fatalf("tenant=%q", got.TenantID)
	}
}
