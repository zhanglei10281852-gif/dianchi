package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/transport"
)

func TestRejectedStopDoesNotAlterPlannedTrip(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	svc := NewTransportService()
	if err := svc.AddVehicle(ctx, p, transport.Vehicle{ID: "truck-6", Plate: "NEW-006", Capacity: 500, HazardApproved: true}); err != nil {
		t.Fatal(err)
	}
	trip := transport.Trip{ID: "trip-6", VehicleID: "truck-6"}
	if err := svc.Plan(ctx, p, trip); err != nil {
		t.Fatal(err)
	}
	bad := transport.Stop{ID: "stop-6", Location: "Depot", Sequence: 0}
	if err := svc.AddStop(ctx, p, trip.ID, bad); err == nil {
		t.Fatal("zero-sequence stop was accepted")
	}
	got, err := svc.Get(ctx, p, trip.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Stops) != 0 {
		t.Fatalf("rejected stop remained on trip: %+v", got.Stops)
	}
	good := transport.Stop{ID: "stop-6", Location: "Depot", Sequence: 1}
	if err := svc.AddStop(ctx, p, trip.ID, good); err != nil {
		t.Fatalf("corrected stop failed: %v", err)
	}
}
