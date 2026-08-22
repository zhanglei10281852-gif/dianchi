package battery

import (
	"fmt"
	"strings"
	"time"
)

type State string

const (
	Received    State = "received"
	Inspected   State = "inspected"
	Reserved    State = "reserved"
	Dismantling State = "dismantling"
	Recovered   State = "recovered"
	Certified   State = "certified"
	Quarantined State = "quarantined"
)

type InspectionResult string

const (
	InspectionPass   InspectionResult = "pass"
	InspectionHold   InspectionResult = "hold"
	InspectionReject InspectionResult = "reject"
)

type Lot struct {
	ID, TenantID, Code, Chemistry string
	State                         State
	Version                       int
	ReceivedAt                    time.Time
	ExpiresAt                     *time.Time
	HazardScore                   int
	CreatedBy                     string
}
type Inspection struct {
	ID, LotID, TenantID, Notes, InspectorID string
	Result                                  InspectionResult
	CreatedAt                               time.Time
}
type Reservation struct {
	ID, LotID, TenantID, Station, OperatorID, State string
	CreatedAt                                       time.Time
	ReleasedAt                                      *time.Time
}
type Recovery struct {
	ID, LotID, TenantID                    string
	LithiumGrams, NickelGrams, CobaltGrams int
	State                                  string
	Version                                int
	RecoveredAt                            time.Time
}
type Certificate struct {
	ID, LotID, TenantID, Serial, State string
	PublishedAt                        *time.Time
	CreatedAt                          time.Time
}

func NormalizeChemistry(v string) (string, error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v != "LFP" && v != "NMC" && v != "LMO" {
		return "", fmt.Errorf("unsupported chemistry %q", v)
	}
	return v, nil
}
func (s State) ValidateAnalyticsEvent() error {
	switch s {
	case Received, Recovered, Quarantined:
		return nil
	default:
		return fmt.Errorf("state %s is not an analytics event", s)
	}
}

func (l Lot) CanInspect() bool { return l.State == Received }
func (l Lot) CanReserve() bool { return l.State == Inspected && l.HazardScore < 70 }
func (l Lot) CanRecover() bool { return l.State == Dismantling }
func (l Lot) CanCertify() bool { return l.State == Recovered }
func NextState(current State, target State) error {
	valid := map[State][]State{Received: {Inspected, Quarantined}, Inspected: {Reserved, Quarantined}, Reserved: {Dismantling}, Dismantling: {Recovered, Quarantined}, Recovered: {Certified}}
	for _, s := range valid[current] {
		if s == target {
			return nil
		}
	}
	return fmt.Errorf("invalid transition %s -> %s", current, target)
}
