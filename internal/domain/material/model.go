package material

import "time"

type Grade string

const (
	GradeA      Grade = "A"
	GradeB      Grade = "B"
	GradeC      Grade = "C"
	GradeReject Grade = "reject"
)

type Kind string

const (
	Lithium  Kind = "lithium"
	Nickel   Kind = "nickel"
	Cobalt   Kind = "cobalt"
	Copper   Kind = "copper"
	Graphite Kind = "graphite"
)

type Lot struct {
	ID, RecoveryID, TenantID string
	Kind                     Kind
	Grams                    int
	Grade                    Grade
	AssayedAt                time.Time
	Assayer                  string
}
type Assay struct {
	ID, RecoveryID, TenantID string
	Kind                     Kind
	Grade                    Grade
	MoisturePPM              int
	PurityPPM                int
	Notes                    string
	Assayer                  string
	AssayedAt                time.Time
}
type Balance struct {
	Kind      Kind
	Incoming  int
	Accepted  int
	Rejected  int
	Available int
}

func (l Lot) IsUsable() bool { return l.Grams > 0 && l.Grade != GradeReject }
func (l Lot) QualityScore() int {
	score := l.Grams / 100
	if l.Grade == GradeA {
		score += 30
	}
	if l.Grade == GradeB {
		score += 15
	}
	if l.Grade == GradeC {
		score += 5
	}
	return score
}
func (a Assay) Passes() bool { return a.PurityPPM >= 950000 && a.MoisturePPM <= 500 }
