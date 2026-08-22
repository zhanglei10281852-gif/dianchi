package dispatch

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	Planned   Status = "planned"
	Loading   Status = "loading"
	EnRoute   Status = "en_route"
	Delivered Status = "delivered"
	Cancelled Status = "cancelled"
)

type Stop struct {
	ID, LotID, Address     string
	Hazard                 int
	WeightKg               int
	WindowStart, WindowEnd time.Time
}

func (s Stop) Validate() error {
	if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.LotID) == "" || strings.TrimSpace(s.Address) == "" {
		return errors.New("stop identity and address are required")
	}
	if s.Hazard < 0 || s.Hazard > 10 {
		return errors.New("hazard must be between zero and ten")
	}
	if s.WeightKg <= 0 {
		return errors.New("stop weight must be positive")
	}
	if s.WindowStart.IsZero() || s.WindowEnd.IsZero() || !s.WindowStart.Before(s.WindowEnd) {
		return errors.New("invalid delivery window")
	}
	return nil
}

type Vehicle struct {
	ID            string
	CapacityKg    int
	HazardLimit   int
	AvailableFrom time.Time
}

func (v Vehicle) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return errors.New("vehicle id is required")
	}
	if v.CapacityKg <= 0 {
		return errors.New("vehicle capacity must be positive")
	}
	if v.HazardLimit < 0 {
		return errors.New("hazard limit must not be negative")
	}
	return nil
}

type Route struct {
	ID, TenantID, VehicleID string
	Stops                   []Stop
	Status                  Status
	TotalWeightKg           int
	MaxHazard               int
	Version                 int
	CreatedAt               time.Time
}

func NewRoute(id, tenant string, v Vehicle, now time.Time) (Route, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(tenant) == "" {
		return Route{}, errors.New("route identity is required")
	}
	if err := v.Validate(); err != nil {
		return Route{}, err
	}
	if now.IsZero() {
		return Route{}, errors.New("route time is required")
	}
	return Route{ID: id, TenantID: tenant, VehicleID: v.ID, Status: Planned, Version: 1, CreatedAt: now}, nil
}
func (r Route) CanAdd() bool { return r.Status == Planned || r.Status == Loading }
func (r Route) AddStop(s Stop, v Vehicle) (Route, error) {
	if !r.CanAdd() {
		return Route{}, fmt.Errorf("route cannot add stop in %s", r.Status)
	}
	if err := s.Validate(); err != nil {
		return Route{}, err
	}
	if r.TotalWeightKg+s.WeightKg > v.CapacityKg {
		return Route{}, errors.New("vehicle capacity exceeded")
	}
	if s.Hazard > v.HazardLimit {
		return Route{}, errors.New("vehicle hazard limit exceeded")
	}
	for _, old := range r.Stops {
		if old.ID == s.ID || old.LotID == s.LotID {
			return Route{}, errors.New("stop already assigned")
		}
	}
	r.Stops = append(append([]Stop(nil), r.Stops...), s)
	r.TotalWeightKg += s.WeightKg
	if s.Hazard > r.MaxHazard {
		r.MaxHazard = s.Hazard
	}
	return r, nil
}
func (r Route) StartLoading() (Route, error) {
	if r.Status != Planned || len(r.Stops) == 0 {
		return Route{}, errors.New("route is not ready for loading")
	}
	r.Status = Loading
	r.Version++
	return r, nil
}
func (r Route) Depart() (Route, error) {
	if r.Status != Loading {
		return Route{}, errors.New("route must be loaded before departure")
	}
	r.Status = EnRoute
	r.Version++
	return r, nil
}
func (r Route) Deliver(now time.Time) (Route, error) {
	if r.Status != EnRoute {
		return Route{}, errors.New("route is not en route")
	}
	if now.IsZero() {
		return Route{}, errors.New("delivery time is required")
	}
	r.Status = Delivered
	r.Version++
	return r, nil
}
func (r Route) Cancel(reason string) (Route, error) {
	if r.Status == Delivered || r.Status == Cancelled {
		return Route{}, errors.New("route cannot be cancelled")
	}
	if strings.TrimSpace(reason) == "" {
		return Route{}, errors.New("cancellation reason is required")
	}
	r.Status = Cancelled
	r.Version++
	return r, nil
}
func (r Route) OrderedStops() []Stop {
	out := append([]Stop(nil), r.Stops...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].WindowStart.Before(out[j].WindowStart) })
	return out
}
func (r Route) Snapshot() Route { r.Stops = append([]Stop(nil), r.Stops...); return r }

type Planner struct {
	mu       sync.Mutex
	routes   map[string]Route
	vehicles map[string]Vehicle
}

func NewPlanner(vehicles []Vehicle) (*Planner, error) {
	p := &Planner{routes: map[string]Route{}, vehicles: map[string]Vehicle{}}
	for _, v := range vehicles {
		if err := v.Validate(); err != nil {
			return nil, err
		}
		if _, ok := p.vehicles[v.ID]; ok {
			return nil, errors.New("duplicate vehicle")
		}
		p.vehicles[v.ID] = v
	}
	return p, nil
}
func (p *Planner) Create(id, tenant, vehicle string, now time.Time) (Route, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.vehicles[vehicle]
	if !ok {
		return Route{}, errors.New("vehicle not found")
	}
	if _, ok = p.routes[id]; ok {
		return Route{}, errors.New("route already exists")
	}
	r, e := NewRoute(id, tenant, v, now)
	if e == nil {
		p.routes[id] = r
	}
	return r, e
}
func (p *Planner) AddStop(id string, s Stop) (Route, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.routes[id]
	if !ok {
		return Route{}, errors.New("route not found")
	}
	v := p.vehicles[r.VehicleID]
	next, e := r.AddStop(s, v)
	if e == nil {
		p.routes[id] = next
	}
	return next.Snapshot(), e
}
func (p *Planner) Transition(id string, next Status, now time.Time) (Route, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.routes[id]
	if !ok {
		return Route{}, errors.New("route not found")
	}
	var e error
	switch next {
	case Loading:
		r, e = r.StartLoading()
	case EnRoute:
		r, e = r.Depart()
	case Delivered:
		r, e = r.Deliver(now)
	default:
		e = errors.New("unsupported transition")
	}
	if e == nil {
		p.routes[id] = r
	}
	return r, e
}
func (p *Planner) Get(id string) (Route, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.routes[id]
	if !ok {
		return Route{}, false
	}
	return r.Snapshot(), true
}
func (p *Planner) List(tenant string) []Route {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Route, 0)
	for _, r := range p.routes {
		if r.TenantID == tenant {
			out = append(out, r.Snapshot())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}
