package reconciliation

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type State string

const (
	Draft         State = "draft"
	Submitted     State = "submitted"
	Investigating State = "investigating"
	Approved      State = "approved"
	Rejected      State = "rejected"
)

type Material string

const (
	Lithium  Material = "lithium"
	Nickel   Material = "nickel"
	Cobalt   Material = "cobalt"
	Copper   Material = "copper"
	Aluminum Material = "aluminum"
)

type Measurement struct {
	LotID      string
	Material   Material
	Expected   int
	Actual     int
	Unit       string
	MeasuredAt time.Time
	Source     string
}

func (m Measurement) Validate() error {
	if strings.TrimSpace(m.LotID) == "" {
		return errors.New("lot id is required")
	}
	if !m.Material.Valid() {
		return fmt.Errorf("unsupported material %q", m.Material)
	}
	if m.Expected < 0 || m.Actual < 0 {
		return errors.New("weights must not be negative")
	}
	if m.Unit != "g" && m.Unit != "kg" {
		return errors.New("unit must be g or kg")
	}
	if m.MeasuredAt.IsZero() {
		return errors.New("measurement time is required")
	}
	if strings.TrimSpace(m.Source) == "" {
		return errors.New("measurement source is required")
	}
	return nil
}

func (m Material) Valid() bool {
	switch m {
	case Lithium, Nickel, Cobalt, Copper, Aluminum:
		return true
	}
	return false
}

type Discrepancy struct {
	LotID           string
	Material        Material
	Expected        int
	Actual          int
	Delta           int
	Ratio           float64
	WithinTolerance bool
}

type Report struct {
	ID            string
	TenantID      string
	State         State
	Measurements  []Measurement
	Discrepancies []Discrepancy
	TotalExpected int
	TotalActual   int
	CreatedAt     time.Time
	SubmittedAt   *time.Time
	ApprovedAt    *time.Time
	ReviewerID    string
	Version       int
}

func NewReport(id, tenant string, now time.Time) (Report, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(tenant) == "" {
		return Report{}, errors.New("report identity is required")
	}
	if now.IsZero() {
		return Report{}, errors.New("report time is required")
	}
	return Report{ID: id, TenantID: tenant, State: Draft, CreatedAt: now, Version: 1}, nil
}

func (r Report) CanAppend() bool  { return r.State == Draft || r.State == Investigating }
func (r Report) CanSubmit() bool  { return r.State == Draft && len(r.Measurements) > 0 }
func (r Report) CanApprove() bool { return r.State == Submitted || r.State == Investigating }

func (r Report) Append(m Measurement) (Report, error) {
	if !r.CanAppend() {
		return Report{}, fmt.Errorf("report %s cannot accept measurements in %s", r.ID, r.State)
	}
	if err := m.Validate(); err != nil {
		return Report{}, err
	}
	for _, old := range r.Measurements {
		if old.LotID == m.LotID && old.Material == m.Material {
			return Report{}, errors.New("duplicate lot material measurement")
		}
	}
	r.Measurements = append(append([]Measurement(nil), r.Measurements...), m)
	r.TotalExpected += normalizeWeight(m.Expected, m.Unit)
	r.TotalActual += normalizeWeight(m.Actual, m.Unit)
	sort.SliceStable(r.Measurements, func(i, j int) bool { return r.Measurements[i].MeasuredAt.Before(r.Measurements[j].MeasuredAt) })
	return r, nil
}

func normalizeWeight(value int, unit string) int {
	if unit == "kg" {
		return value * 1000
	}
	return value
}

func (r Report) Evaluate(tolerance float64) (Report, error) {
	if tolerance < 0 || tolerance > 1 {
		return Report{}, errors.New("tolerance must be between zero and one")
	}
	if len(r.Measurements) == 0 {
		return Report{}, errors.New("report has no measurements")
	}
	r.Discrepancies = make([]Discrepancy, 0, len(r.Measurements))
	for _, m := range r.Measurements {
		e, a := normalizeWeight(m.Expected, m.Unit), normalizeWeight(m.Actual, m.Unit)
		d := e - a
		ratio := 0.0
		if e > 0 {
			ratio = float64(abs(d)) / float64(e)
		}
		r.Discrepancies = append(r.Discrepancies, Discrepancy{LotID: m.LotID, Material: m.Material, Expected: e, Actual: a, Delta: d, Ratio: ratio, WithinTolerance: ratio <= tolerance})
	}
	return r, nil
}

func (r Report) Submit(now time.Time) (Report, error) {
	if !r.CanSubmit() {
		return Report{}, fmt.Errorf("report cannot be submitted in %s", r.State)
	}
	if now.IsZero() {
		return Report{}, errors.New("submission time is required")
	}
	r.State, r.SubmittedAt = Submitted, &now
	r.Version++
	return r, nil
}

func (r Report) StartInvestigation() (Report, error) {
	if r.State != Submitted {
		return Report{}, fmt.Errorf("report cannot be investigated in %s", r.State)
	}
	r.State, r.Version = Investigating, r.Version+1
	return r, nil
}

func (r Report) Approve(reviewer string, now time.Time) (Report, error) {
	if !r.CanApprove() {
		return Report{}, fmt.Errorf("report cannot be approved in %s", r.State)
	}
	if strings.TrimSpace(reviewer) == "" {
		return Report{}, errors.New("reviewer is required")
	}
	if now.IsZero() {
		return Report{}, errors.New("approval time is required")
	}
	for _, d := range r.Discrepancies {
		if !d.WithinTolerance {
			return Report{}, errors.New("material discrepancy requires investigation")
		}
	}
	r.State, r.ReviewerID, r.ApprovedAt, r.Version = Approved, reviewer, &now, r.Version+1
	return r, nil
}

func (r Report) Reject(reviewer string) (Report, error) {
	if !r.CanApprove() {
		return Report{}, fmt.Errorf("report cannot be rejected in %s", r.State)
	}
	if strings.TrimSpace(reviewer) == "" {
		return Report{}, errors.New("reviewer is required")
	}
	r.State, r.ReviewerID, r.Version = Rejected, reviewer, r.Version+1
	return r, nil
}

func (r Report) Snapshot() Report {
	r.Measurements = append([]Measurement(nil), r.Measurements...)
	r.Discrepancies = append([]Discrepancy(nil), r.Discrepancies...)
	return r
}

func (r Report) DurableSnapshot() Report {
	snapshot := r.Snapshot()
	snapshot.Measurements = append([]Measurement(nil), snapshot.Measurements...)
	snapshot.Discrepancies = append([]Discrepancy(nil), snapshot.Discrepancies...)
	return snapshot
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
