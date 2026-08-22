package dispatch

import (
	"strings"
	"testing"
	"time"
)

func TestStopValidationMessages(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		stop Stop
		want string
	}{
		{"identity", Stop{WindowStart: now, WindowEnd: now.Add(time.Hour)}, "identity"},
		{"hazard", Stop{ID: "s", LotID: "l", Address: "a", Hazard: 11, WeightKg: 1, WindowStart: now, WindowEnd: now.Add(time.Hour)}, "hazard"},
		{"weight", Stop{ID: "s", LotID: "l", Address: "a", Hazard: 1, WeightKg: 0, WindowStart: now, WindowEnd: now.Add(time.Hour)}, "weight"},
		{"window", Stop{ID: "s", LotID: "l", Address: "a", Hazard: 1, WeightKg: 1, WindowStart: now, WindowEnd: now}, "window"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.stop.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v want %q", err, tc.want)
			}
		})
	}
}

func TestVehicleValidation(t *testing.T) {
	cases := []Vehicle{
		{},
		{ID: "v", CapacityKg: 0},
		{ID: "v", CapacityKg: 10, HazardLimit: -1},
	}
	for _, v := range cases {
		if err := v.Validate(); err == nil {
			t.Fatalf("invalid vehicle accepted: %+v", v)
		}
	}
	if err := (Vehicle{ID: "v", CapacityKg: 10}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRouteCancellationStateGuards(t *testing.T) {
	now := time.Now()
	r, _ := NewRoute("r", "t", vehicle(), now)
	r, _ = r.AddStop(stop("s", 1, 2, now), vehicle())
	r, _ = r.StartLoading()
	r, _ = r.Depart()
	r, _ = r.Deliver(now.Add(time.Hour))
	if _, err := r.Cancel("late cancellation"); err == nil {
		t.Fatal("delivered route cancelled")
	}
	if _, err := r.Deliver(now); err == nil {
		t.Fatal("delivered route delivered twice")
	}
}

func TestPlannerDuplicateVehicleAndRoute(t *testing.T) {
	v := vehicle()
	if _, err := NewPlanner([]Vehicle{v, v}); err == nil {
		t.Fatal("duplicate vehicle accepted")
	}
	p, err := NewPlanner([]Vehicle{v})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Create("r", "t", v.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Create("r", "t", v.ID, time.Now()); err == nil {
		t.Fatal("duplicate route accepted")
	}
}

func TestPlannerTransitionUnknownRouteAndStatus(t *testing.T) {
	p, _ := NewPlanner([]Vehicle{vehicle()})
	if _, err := p.Transition("missing", Loading, time.Now()); err == nil {
		t.Fatal("unknown route transitioned")
	}
	if _, err := p.Create("r", "t", "truck-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Transition("r", Status("unknown"), time.Now()); err == nil {
		t.Fatal("unknown transition accepted")
	}
}

func TestRouteSnapshotCopiesStops(t *testing.T) {
	now := time.Now()
	r, _ := NewRoute("r", "tenant", vehicle(), now)
	r, _ = r.AddStop(stop("s", 1, 2, now), vehicle())
	snapshot := r.Snapshot()
	snapshot.Stops[0].Address = "changed"
	if r.Stops[0].Address == "changed" {
		t.Fatal("route snapshot shared stop storage")
	}
}

func TestRouteVersionOnlyChangesOnSuccessfulTransition(t *testing.T) {
	now := time.Now()
	r, _ := NewRoute("r", "tenant", vehicle(), now)
	version := r.Version
	if _, err := r.Deliver(now); err == nil {
		t.Fatal("invalid delivery succeeded")
	}
	if r.Version != version {
		t.Fatalf("failed transition changed version from %d to %d", version, r.Version)
	}
	r, _ = r.AddStop(stop("s", 1, 2, now), vehicle())
	r, _ = r.StartLoading()
	if r.Version != version+1 {
		t.Fatalf("loading version=%d", r.Version)
	}
}

func TestPlannerListReturnsCreationOrder(t *testing.T) {
	now := time.Now()
	p, _ := NewPlanner([]Vehicle{vehicle()})
	if _, err := p.Create("later", "tenant", "truck-1", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Create("first", "tenant", "truck-1", now); err != nil {
		t.Fatal(err)
	}
	routes := p.List("tenant")
	if len(routes) != 2 || routes[0].ID != "first" || routes[1].ID != "later" {
		t.Fatalf("routes=%+v", routes)
	}
}

func TestRouteAddStopLeavesOriginalUnchangedOnFailure(t *testing.T) {
	now := time.Now()
	v := vehicle()
	r, _ := NewRoute("route", "tenant", v, now)
	r, _ = r.AddStop(stop("accepted", 1, 90, now), v)
	before := r.Snapshot()
	if _, err := r.AddStop(stop("too-heavy", 1, 20, now), v); err == nil {
		t.Fatal("over-capacity stop accepted")
	}
	if r.TotalWeightKg != before.TotalWeightKg || len(r.Stops) != len(before.Stops) {
		t.Fatalf("failed add mutated route: before=%+v after=%+v", before, r)
	}
}
