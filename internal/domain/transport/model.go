package transport

import "time"

type Status string

const (
	Planned   Status = "planned"
	Loading   Status = "loading"
	InTransit Status = "in_transit"
	Arrived   Status = "arrived"
	Closed    Status = "closed"
)

type Vehicle struct {
	ID, Plate, Carrier string
	Capacity           int
	HazardApproved     bool
}
type Trip struct {
	ID, TenantID, ManifestID, VehicleID, Driver string
	Status                                      Status
	DepartedAt, ArrivedAt                       *time.Time
	Stops                                       []Stop
}
type Stop struct {
	ID, TripID, Location string
	Sequence             int
	ArrivedAt            *time.Time
	Temperature          float64
}

func (t Trip) CanLoad() bool   { return t.Status == Planned }
func (t Trip) CanDepart() bool { return t.Status == Loading && len(t.Stops) >= 2 }
func (t Trip) CanClose() bool  { return t.Status == Arrived }
func (t Trip) DistanceStops() int {
	n := 0
	for _, s := range t.Stops {
		n += s.Sequence
	}
	return n
}
