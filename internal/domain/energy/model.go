package energy

import "time"

type Reading struct {
	ID, TenantID, LotID           string
	Voltage, Current, Temperature float64
	At                            time.Time
}
type Window struct {
	Start, End time.Time
	Peak       bool
}

func (r Reading) Power() float64           { return r.Voltage * r.Current }
func (r Reading) Safe() bool               { return r.Voltage > 0 && r.Temperature > -20 && r.Temperature < 60 }
func (w Window) Contains(t time.Time) bool { return !t.Before(w.Start) && t.Before(w.End) }
