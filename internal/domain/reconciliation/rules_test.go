package reconciliation

import (
	"strings"
	"testing"
	"time"
)

func TestMaterialValidityTable(t *testing.T) {
	cases := []struct {
		material Material
		valid    bool
	}{
		{Lithium, true},
		{Nickel, true},
		{Cobalt, true},
		{Copper, true},
		{Aluminum, true},
		{Material("graphite"), false},
	}
	for _, tc := range cases {
		if got := tc.material.Valid(); got != tc.valid {
			t.Errorf("material %q valid=%v want %v", tc.material, got, tc.valid)
		}
	}
}

func TestMeasurementValidationTable(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		m    Measurement
		want string
	}{
		{"missing lot", Measurement{Material: Lithium, Expected: 1, Actual: 1, Unit: "g", MeasuredAt: now, Source: "scale"}, "lot id"},
		{"unknown material", Measurement{LotID: "lot", Material: "glass", Expected: 1, Actual: 1, Unit: "g", MeasuredAt: now, Source: "scale"}, "unsupported"},
		{"negative expected", Measurement{LotID: "lot", Material: Lithium, Expected: -1, Actual: 1, Unit: "g", MeasuredAt: now, Source: "scale"}, "negative"},
		{"negative actual", Measurement{LotID: "lot", Material: Lithium, Expected: 1, Actual: -1, Unit: "g", MeasuredAt: now, Source: "scale"}, "negative"},
		{"bad unit", Measurement{LotID: "lot", Material: Lithium, Expected: 1, Actual: 1, Unit: "lb", MeasuredAt: now, Source: "scale"}, "unit"},
		{"missing time", Measurement{LotID: "lot", Material: Lithium, Expected: 1, Actual: 1, Unit: "g", Source: "scale"}, "time"},
		{"missing source", Measurement{LotID: "lot", Material: Lithium, Expected: 1, Actual: 1, Unit: "g", MeasuredAt: now}, "source"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.m.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want substring %q", err, tc.want)
			}
		})
	}
}

func TestReportConstructionValidation(t *testing.T) {
	now := time.Now()
	cases := []struct {
		id, tenant string
		at         time.Time
		valid      bool
	}{
		{"", "tenant", now, false},
		{"report", "", now, false},
		{"report", "tenant", time.Time{}, false},
		{"report", "tenant", now, true},
	}
	for _, tc := range cases {
		r, err := NewReport(tc.id, tc.tenant, tc.at)
		if tc.valid && (err != nil || r.State != Draft || r.Version != 1) {
			t.Fatalf("valid construction report=%+v err=%v", r, err)
		}
		if !tc.valid && err == nil {
			t.Fatalf("invalid construction accepted: %+v", tc)
		}
	}
}

func TestReportStateGuards(t *testing.T) {
	now := time.Now()
	r, err := NewReport("r", "t", now)
	if _, err := r.StartInvestigation(); err == nil {
		t.Fatal("draft investigation accepted")
	}
	if _, err := r.Approve("reviewer", now); err == nil {
		t.Fatal("draft approval accepted")
	}
	if _, err := r.Reject("reviewer"); err == nil {
		t.Fatal("draft rejection accepted")
	}
	r, _ = r.Append(measurement(now, "lot", Lithium, 10, 10, "g"))
	r, _ = r.Submit(now)
	if _, err := r.Submit(now); err == nil {
		t.Fatal("double submit accepted")
	}
	r, err = r.StartInvestigation()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.StartInvestigation(); err == nil {
		t.Fatal("double investigation accepted")
	}
}

func TestEvaluateZeroExpectedAndOrdering(t *testing.T) {
	now := time.Now()
	r, _ := NewReport("r", "t", now)
	r, _ = r.Append(measurement(now, "z", Lithium, 0, 10, "g"))
	r, _ = r.Append(measurement(now, "a", Nickel, 100, 100, "g"))
	out, err := EvaluateWithPolicy(now, DefaultPolicy(), r)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Discrepancies) != 2 || out.Discrepancies[0].LotID != "a" {
		t.Fatalf("discrepancies=%+v", out.Discrepancies)
	}
	if out.Discrepancies[1].Ratio != 0 {
		t.Fatalf("zero expected ratio=%v", out.Discrepancies[1].Ratio)
	}
}

func TestPolicyToleranceErrors(t *testing.T) {
	p := DefaultPolicy()
	if _, err := p.Tolerance("glass"); err == nil {
		t.Fatal("unknown policy accepted")
	}
	p.ToleranceByMaterial[Lithium] = 2
	if _, err := p.Tolerance(Lithium); err == nil {
		t.Fatal("out of range policy accepted")
	}
	p = DefaultPolicy()
	p.ToleranceByMaterial = map[Material]float64{}
	if _, err := p.Tolerance(Lithium); err == nil {
		t.Fatal("missing policy accepted")
	}
}

func TestPolicyTimeBoundaries(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	r, _ := NewReport("r", "t", now)
	r, _ = r.Append(measurement(now.Add(-72*time.Hour), "old", Lithium, 1, 1, "g"))
	if _, err := EvaluateWithPolicy(now, DefaultPolicy(), r); err != nil {
		t.Fatalf("boundary measurement rejected: %v", err)
	}
	r, _ = NewReport("r2", "t", now)
	r, _ = r.Append(measurement(now.Add(5*time.Minute), "future", Lithium, 1, 1, "g"))
	if _, err := EvaluateWithPolicy(now, DefaultPolicy(), r); err != nil {
		t.Fatalf("future boundary rejected: %v", err)
	}
}

func TestDigestIncludesMeasurementFacts(t *testing.T) {
	now := time.Now()
	a, _ := NewReport("a", "t", now)
	a, _ = a.Append(measurement(now, "lot", Lithium, 10, 9, "g"))
	b, _ := NewReport("b", "t", now)
	b, _ = b.Append(measurement(now, "lot", Lithium, 10, 8, "g"))
	if CanonicalDigest(a) == CanonicalDigest(b) {
		t.Fatal("actual weight omitted from digest")
	}
	c, _ := NewReport("c", "t", now)
	c, _ = c.Append(measurement(now.Add(time.Second), "lot", Lithium, 10, 9, "g"))
	if CanonicalDigest(a) == CanonicalDigest(c) {
		t.Fatal("measurement time omitted from digest")
	}
}

func TestAppendSortsMeasurementsWithoutSharingBackingArray(t *testing.T) {
	now := time.Now()
	r, _ := NewReport("r", "t", now)
	r, _ = r.Append(measurement(now.Add(2*time.Minute), "late", Lithium, 1, 1, "g"))
	first := r.Measurements
	r, _ = r.Append(measurement(now, "early", Lithium, 1, 1, "g"))
	if len(first) != 1 || first[0].LotID != "late" {
		t.Fatalf("old slice changed: %+v", first)
	}
	if r.Measurements[0].LotID != "early" {
		t.Fatalf("not sorted: %+v", r.Measurements)
	}
}

func TestReportEvaluateDoesNotMutateSource(t *testing.T) {
	now := time.Now()
	r, _ := NewReport("r", "t", now)
	r, _ = r.Append(measurement(now, "lot", Lithium, 10, 9, "g"))
	if len(r.Discrepancies) != 0 {
		t.Fatal("append unexpectedly evaluated")
	}
	copy := r
	copy, _ = EvaluateWithPolicy(now, DefaultPolicy(), copy)
	if len(r.Discrepancies) != 0 || len(copy.Discrepancies) != 1 {
		t.Fatalf("source=%+v evaluated=%+v", r, copy)
	}
}
