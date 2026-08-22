package reconciliation

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Policy struct {
	ToleranceByMaterial  map[Material]float64
	RequireInvestigation bool
	MaxMeasurementAge    time.Duration
}

func DefaultPolicy() Policy {
	return Policy{ToleranceByMaterial: map[Material]float64{Lithium: .05, Nickel: .08, Cobalt: .05, Copper: .1, Aluminum: .12}, RequireInvestigation: true, MaxMeasurementAge: 72 * time.Hour}
}

func (p Policy) Tolerance(m Material) (float64, error) {
	if !m.Valid() {
		return 0, errors.New("invalid material")
	}
	v, ok := p.ToleranceByMaterial[m]
	if !ok {
		return 0, fmt.Errorf("no policy for %s", m)
	}
	if v < 0 || v > 1 {
		return 0, errors.New("policy tolerance out of range")
	}
	return v, nil
}

func (p Policy) Validate(now time.Time, report Report) error {
	if now.IsZero() {
		return errors.New("current time is required")
	}
	if p.MaxMeasurementAge <= 0 {
		return errors.New("measurement age must be positive")
	}
	for _, m := range report.Measurements {
		if now.Sub(m.MeasuredAt) > p.MaxMeasurementAge {
			return fmt.Errorf("measurement for %s is too old", m.LotID)
		}
		if m.MeasuredAt.After(now.Add(5 * time.Minute)) {
			return fmt.Errorf("measurement for %s is in the future", m.LotID)
		}
	}
	return nil
}

func EvaluateWithPolicy(now time.Time, p Policy, r Report) (Report, error) {
	if err := p.Validate(now, r); err != nil {
		return Report{}, err
	}
	result := r.Snapshot()
	result.Discrepancies = make([]Discrepancy, 0, len(result.Measurements))
	for _, m := range result.Measurements {
		tol, err := p.Tolerance(m.Material)
		if err != nil {
			return Report{}, err
		}
		e, a := normalizeWeight(m.Expected, m.Unit), normalizeWeight(m.Actual, m.Unit)
		delta := e - a
		ratio := 0.0
		if e > 0 {
			ratio = float64(abs(delta)) / float64(e)
		}
		result.Discrepancies = append(result.Discrepancies, Discrepancy{LotID: m.LotID, Material: m.Material, Expected: e, Actual: a, Delta: delta, Ratio: ratio, WithinTolerance: ratio <= tol})
	}
	sort.Slice(result.Discrepancies, func(i, j int) bool { return result.Discrepancies[i].LotID < result.Discrepancies[j].LotID })
	return result, nil
}

func NeedsInvestigation(r Report) bool {
	for _, d := range r.Discrepancies {
		if !d.WithinTolerance {
			return true
		}
	}
	return false
}

func CanonicalDigest(r Report) string {
	parts := make([]string, 0, len(r.Measurements))
	for _, m := range r.Measurements {
		parts = append(parts, fmt.Sprintf("%s|%s|%d|%d|%s|%d", m.LotID, m.Material, m.Expected, m.Actual, m.Unit, m.MeasuredAt.UnixNano()))
	}
	sort.Strings(parts)
	h := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(h[:])
}

func Summarize(r Report) map[Material]int {
	out := make(map[Material]int)
	for _, m := range r.Measurements {
		out[m.Material] += normalizeWeight(m.Actual, m.Unit)
	}
	return out
}
