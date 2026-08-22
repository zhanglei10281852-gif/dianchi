package incident

import "time"

type Severity string

const (
	Low      Severity = "low"
	Medium   Severity = "medium"
	High     Severity = "high"
	Critical Severity = "critical"
)

type Status string

const (
	Open          Status = "open"
	Investigating Status = "investigating"
	Contained     Status = "contained"
	Resolved      Status = "resolved"
)

type Incident struct {
	ID, TenantID, LotID, Reporter, Summary string
	Severity                               Severity
	Status                                 Status
	OpenedAt                               time.Time
	ClosedAt                               *time.Time
}
type Action struct {
	ID, IncidentID, ActorID, Description string
	CreatedAt                            time.Time
}

func (i Incident) CanClose() bool  { return i.Status == Contained }
func (i Incident) Escalated() bool { return i.Severity == High || i.Severity == Critical }
