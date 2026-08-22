package operator

import "time"

type Shift string

const (
	Day    Shift = "day"
	Night  Shift = "night"
	OnCall Shift = "on_call"
)

type Certification string

const (
	Safety     Certification = "safety"
	Forklift   Certification = "forklift"
	Hazmat     Certification = "hazmat"
	Supervisor Certification = "supervisor"
)

type Operator struct {
	ID, TenantID, Name string
	Shift              Shift
	Certifications     map[Certification]time.Time
	Active             bool
}

func (o Operator) Certified(c Certification, now time.Time) bool {
	at, ok := o.Certifications[c]
	return ok && o.Active && now.Before(at)
}
func (o Operator) CanHandleHazard(now time.Time) bool {
	return o.Certified(Safety, now) && o.Certified(Hazmat, now)
}
