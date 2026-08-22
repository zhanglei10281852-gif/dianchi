package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/transport"
	"testing"
)

func TestEcoTransportStopValidation(t *testing.T) {
	s := NewTransportService()
	p := auth.Principal{ID: "o", TenantID: "e", Role: auth.Operator}
	if e := s.AddVehicle(context.Background(), p, transport.Vehicle{ID: "v", Capacity: 100}); e != nil {
		t.Fatal(e)
	}
	if e := s.Plan(context.Background(), p, transport.Trip{ID: "t", VehicleID: "v"}); e != nil {
		t.Fatal(e)
	}
	if e := s.AddStop(context.Background(), p, "t", transport.Stop{ID: "bad"}); e == nil {
		t.Fatal("accepted")
	}
	got, _ := s.Get(context.Background(), p, "t")
	if len(got.Stops) != 0 {
		t.Fatal("persisted")
	}
}
