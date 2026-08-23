package service

import (
	"context"
	"strings"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/transport"
)

func TestTransportServiceAddStopInvalidSequenceNotPersisted(t *testing.T) {
	svc := NewTransportService()
	op := auth.Principal{ID: "op", TenantID: "t", Role: auth.Operator}
	ctx := context.Background()

	veh := transport.Vehicle{ID: "truck", Plate: "P", Carrier: "c", Capacity: 10, HazardApproved: true}
	if err := svc.AddVehicle(ctx, op, veh); err != nil {
		t.Fatal(err)
	}
	if err := svc.Plan(ctx, op, transport.Trip{ID: "trip", VehicleID: "truck", Driver: "d"}); err != nil {
		t.Fatal(err)
	}

	// Sequence zero is invalid; the rejection must leave the trip collection untouched.
	invalid := transport.Stop{ID: "stop", Location: "site", Sequence: 0}
	if err := svc.AddStop(ctx, op, "trip", invalid); err == nil || !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("err=%v, want sequence invalid", err)
	}
	got, err := svc.Get(ctx, op, "trip")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Stops) != 0 {
		t.Fatalf("invalid stop persisted: stops=%+v", got.Stops)
	}

	// Re-adding with a valid, distinct sequence must succeed (no false duplicate).
	valid := transport.Stop{ID: "stop", Location: "site", Sequence: 1}
	if err := svc.AddStop(ctx, op, "trip", valid); err != nil {
		t.Fatalf("retry with valid sequence failed: %v", err)
	}
	got, _ = svc.Get(ctx, op, "trip")
	if len(got.Stops) != 1 || got.Stops[0].Sequence != 1 {
		t.Fatalf("valid stop not added: stops=%+v", got.Stops)
	}
}

func TestTransportServiceAddStopDuplicateNotPersisted(t *testing.T) {
	svc := NewTransportService()
	op := auth.Principal{ID: "op", TenantID: "t", Role: auth.Operator}
	ctx := context.Background()
	veh := transport.Vehicle{ID: "truck", Plate: "P", Carrier: "c", Capacity: 10, HazardApproved: true}
	svc.AddVehicle(ctx, op, veh)
	svc.Plan(ctx, op, transport.Trip{ID: "trip", VehicleID: "truck", Driver: "d"})

	svc.AddStop(ctx, op, "trip", transport.Stop{ID: "s1", Location: "a", Sequence: 1})
	dup := transport.Stop{ID: "s2", Location: "b", Sequence: 1} // duplicate sequence
	if err := svc.AddStop(ctx, op, "trip", dup); err == nil || !strings.Contains(err.Error(), "exists") {
		t.Fatalf("err=%v, want duplicate", err)
	}
	got, _ := svc.Get(ctx, op, "trip")
	if len(got.Stops) != 1 {
		t.Fatalf("duplicate stop persisted: stops=%+v", got.Stops)
	}
}
