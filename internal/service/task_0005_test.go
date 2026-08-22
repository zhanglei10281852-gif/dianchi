package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/transport"
)

func TestRejectedTripMovePreservesLifecycle(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	svc := NewTransportService()
	if err := svc.AddVehicle(ctx, p, transport.Vehicle{ID: "truck-5", Plate: "NEW-005", Capacity: 1000, HazardApproved: true}); err != nil {
		t.Fatal(err)
	}
	trip := transport.Trip{ID: "trip-5", VehicleID: "truck-5"}
	if err := svc.Plan(ctx, p, trip); err != nil {
		t.Fatal(err)
	}
	if err := svc.Move(ctx, p, trip.ID, transport.InTransit); err == nil {
		t.Fatal("planned trip skipped loading")
	}
	got, err := svc.Get(ctx, p, trip.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != transport.Planned || got.DepartedAt != nil {
		t.Fatalf("rejected departure changed trip: %+v", got)
	}
	if err := svc.Move(ctx, p, trip.ID, transport.Loading); err != nil {
		t.Fatalf("legal loading failed: %v", err)
	}
}
