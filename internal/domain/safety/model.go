package safety

import "time"

type Level string

const (
	Clear  Level = "clear"
	Watch  Level = "watch"
	Danger Level = "danger"
)

type Finding struct {
	ID, LotID, TenantID, Code, Description string
	Level                                  Level
	Measured                               float64
	Limit                                  float64
	ObservedAt                             time.Time
	ResolvedAt                             *time.Time
}
type Checklist struct {
	ID, LotID, InspectorID string
	Findings               []Finding
	CompletedAt            *time.Time
}

func (f Finding) Open() bool { return f.ResolvedAt == nil }
func (c Checklist) Passes() bool {
	if c.CompletedAt == nil {
		return false
	}
	for _, f := range c.Findings {
		if f.Open() && f.Level == Danger {
			return false
		}
	}
	return true
}
