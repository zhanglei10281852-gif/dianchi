package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/transport"
	"testing"
)

func TestEcoTransportPlanRequiresVehicle(t *testing.T) {
	s := NewTransportService()
	p := auth.Principal{ID: "o", TenantID: "e", Role: auth.Operator}
	if e := s.Plan(context.Background(), p, transport.Trip{ID: "t", VehicleID: "missing"}); e == nil {
		t.Fatal("accepted")
	}
	if _, e := s.Get(context.Background(), p, "t"); e == nil {
		t.Fatal("persisted")
	}
}
