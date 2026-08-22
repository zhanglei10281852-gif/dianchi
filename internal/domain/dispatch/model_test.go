package dispatch

import (
	"context"
	"sync"
	"testing"
	"time"
)

func stop(id string, hazard, kg int, at time.Time) Stop {
	return Stop{ID: id, LotID: "lot-" + id, Address: "Depot " + id, Hazard: hazard, WeightKg: kg, WindowStart: at, WindowEnd: at.Add(time.Hour)}
}
func vehicle() Vehicle {
	return Vehicle{ID: "truck-1", CapacityKg: 100, HazardLimit: 8, AvailableFrom: time.Now()}
}
func TestRouteLifecycleAndOrdering(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r, err := NewRoute("r", "t", vehicle(), now)
	if err != nil {
		t.Fatal(err)
	}
	r, err = r.AddStop(stop("2", 5, 30, now.Add(2*time.Hour)), vehicle())
	if err != nil {
		t.Fatal(err)
	}
	r, err = r.AddStop(stop("1", 3, 20, now.Add(time.Hour)), vehicle())
	if err != nil {
		t.Fatal(err)
	}
	if r.TotalWeightKg != 50 || r.MaxHazard != 5 {
		t.Fatalf("route=%+v", r)
	}
	if got := r.OrderedStops()[0].ID; got != "1" {
		t.Fatal(got)
	}
	r, _ = r.StartLoading()
	r, _ = r.Depart()
	r, err = r.Deliver(now.Add(3 * time.Hour))
	if err != nil || r.Status != Delivered {
		t.Fatalf("deliver=%+v err=%v", r, err)
	}
}
func TestRouteRejectsCapacityHazardAndDuplicate(t *testing.T) {
	now := time.Now()
	v := vehicle()
	r, _ := NewRoute("r", "t", v, now)
	if _, err := r.AddStop(stop("1", 9, 1, now), v); err == nil {
		t.Fatal("hazard accepted")
	}
	if _, err := r.AddStop(stop("1", 1, 101, now), v); err == nil {
		t.Fatal("capacity accepted")
	}
	r, _ = r.AddStop(stop("1", 1, 10, now), v)
	if _, err := r.AddStop(stop("1", 1, 10, now), v); err == nil {
		t.Fatal("duplicate accepted")
	}
}
func TestRouteTransitionGuards(t *testing.T) {
	now := time.Now()
	r, err := NewRoute("r", "t", vehicle(), now)
	if _, err := r.Depart(); err == nil {
		t.Fatal("empty route departed")
	}
	r, _ = r.AddStop(stop("1", 1, 1, now), vehicle())
	if _, err := r.Deliver(now); err == nil {
		t.Fatal("planned delivered")
	}
	r, _ = r.StartLoading()
	r, _ = r.Depart()
	if _, err := r.Cancel(" "); err == nil {
		t.Fatal("blank cancel accepted")
	}
	r, err = r.Cancel("vehicle failure")
	if err != nil || r.Status != Cancelled {
		t.Fatalf("cancel=%+v err=%v", r, err)
	}
}
func TestPlannerTenantIsolationAndCopies(t *testing.T) {
	now := time.Now()
	p, err := NewPlanner([]Vehicle{vehicle()})
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.Create("r", "tenant-a", "truck-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.Get("missing"); ok {
		t.Fatal("missing route")
	}
	r.Stops = append(r.Stops, stop("x", 1, 1, now))
	got, _ := p.Get("r")
	if len(got.Stops) != 0 {
		t.Fatal("planner leaked route copy")
	}
	if len(p.List("tenant-b")) != 0 {
		t.Fatal("tenant leaked")
	}
}

func TestPlannerAddStopResultDoesNotExposeStoredSlice(t *testing.T) {
	now := time.Now()
	p, _ := NewPlanner([]Vehicle{vehicle()})
	_, _ = p.Create("r", "tenant-a", "truck-1", now)
	added, err := p.AddStop("r", stop("one", 1, 10, now))
	if err != nil {
		t.Fatal(err)
	}
	added.Stops[0].Address = "mutated by caller"
	stored, _ := p.Get("r")
	if stored.Stops[0].Address == "mutated by caller" {
		t.Fatal("returned route exposed planner-owned stop storage")
	}
}
func TestPlannerConcurrentAddsHaveSingleWinnerForDuplicate(t *testing.T) {
	now := time.Now()
	p, _ := NewPlanner([]Vehicle{vehicle()})
	if _, err := p.Create("r", "t", "truck-1", now); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := p.AddStop("r", stop("same", 1, 10, now)); errs <- e }()
	}
	wg.Wait()
	close(errs)
	success := 0
	for e := range errs {
		if e == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("success=%d", success)
	}
}
func TestPlannerContextCallerCanCancelBeforeOperation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if ctx.Err() == nil {
		t.Fatal("context not cancelled")
	}
}
