package reconciliation

import (
	"strings"
	"testing"
	"time"
)

func measurement(at time.Time, lot string, mat Material, expected, actual int, unit string) Measurement {
	return Measurement{LotID: lot, Material: mat, Expected: expected, Actual: actual, Unit: unit, MeasuredAt: at, Source: "scale-1"}
}

func TestReportLifecycleKeepsSnapshotsIndependent(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	r, err := NewReport("r-1", "tenant-a", now)
	if err != nil {
		t.Fatal(err)
	}
	r, err = r.Append(measurement(now, "lot-1", Lithium, 1000, 960, "g"))
	if err != nil {
		t.Fatal(err)
	}
	copy := r.Snapshot()
	copy.Measurements[0].Actual = 1
	if r.Measurements[0].Actual != 960 {
		t.Fatal("snapshot mutated original")
	}
	r, err = r.Submit(now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if r.State != Submitted || r.SubmittedAt == nil {
		t.Fatalf("submit=%+v", r)
	}
	if _, err = r.Append(measurement(now, "lot-2", Nickel, 1, 1, "kg")); err == nil {
		t.Fatal("submitted report accepted measurement")
	}
}
func TestReportRejectsInvalidMeasurementAndDuplicates(t *testing.T) {
	now := time.Now()
	r, _ := NewReport("r", "t", now)
	bad := measurement(now, "", Lithium, 1, 1, "g")
	if _, err := r.Append(bad); err == nil {
		t.Fatal("empty lot accepted")
	}
	r, _ = r.Append(measurement(now, "l", Lithium, 1, 1, "g"))
	if _, err := r.Append(measurement(now, "l", Lithium, 1, 1, "g")); err == nil {
		t.Fatal("duplicate accepted")
	}
}
func TestPolicyUsesMaterialSpecificToleranceAndUnits(t *testing.T) {
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	r, _ := NewReport("r", "t", now)
	var err error
	r, err = r.Append(measurement(now, "l1", Lithium, 1, 950, "kg"))
	if err != nil {
		t.Fatal(err)
	}
	r, err = r.Append(measurement(now, "l2", Nickel, 1000, 930, "g"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := EvaluateWithPolicy(now, DefaultPolicy(), r)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Discrepancies) != 2 {
		t.Fatal(len(out.Discrepancies))
	}
	if out.Discrepancies[0].WithinTolerance {
		t.Fatal("lithium discrepancy should fail")
	}
}
func TestPolicyRejectsStaleAndFutureMeasurements(t *testing.T) {
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	r, _ := NewReport("r", "t", now)
	r, _ = r.Append(measurement(now.Add(-73*time.Hour), "l", Lithium, 1, 1, "g"))
	if _, err := EvaluateWithPolicy(now, DefaultPolicy(), r); err == nil || !strings.Contains(err.Error(), "too old") {
		t.Fatalf("stale err=%v", err)
	}
	r, _ = NewReport("r2", "t", now)
	r, _ = r.Append(measurement(now.Add(6*time.Minute), "l", Lithium, 1, 1, "g"))
	if _, err := EvaluateWithPolicy(now, DefaultPolicy(), r); err == nil || !strings.Contains(err.Error(), "future") {
		t.Fatalf("future err=%v", err)
	}
}
func TestInvestigationAndApprovalRequireCleanDiscrepancy(t *testing.T) {
	now := time.Now()
	r, err := NewReport("r", "t", now)
	r, _ = r.Append(measurement(now, "l", Lithium, 100, 50, "g"))
	r, _ = EvaluateWithPolicy(now, DefaultPolicy(), r)
	r, _ = r.Submit(now)
	r, _ = r.StartInvestigation()
	if _, err := r.Approve("sup", now); err == nil {
		t.Fatal("dirty report approved")
	}
	r.Discrepancies[0].WithinTolerance = true
	r, err = r.Approve("sup", now)
	if err != nil || r.State != Approved {
		t.Fatalf("approve=%+v err=%v", r, err)
	}
}
func TestRejectionAndValidation(t *testing.T) {
	now := time.Now()
	r, err := NewReport("r", "t", now)
	if _, err := r.Submit(now); err == nil {
		t.Fatal("empty submitted")
	}
	r, _ = r.Append(measurement(now, "l", Cobalt, 1, 1, "g"))
	r, _ = r.Submit(now)
	r, err = r.Reject("sup")
	if err != nil || r.State != Rejected {
		t.Fatalf("reject=%+v err=%v", r, err)
	}
	if _, err = r.Reject("sup"); err == nil {
		t.Fatal("rejected report changed")
	}
}
func TestCanonicalDigestStableAcrossOrder(t *testing.T) {
	now := time.Now()
	a, _ := NewReport("a", "t", now)
	a, _ = a.Append(measurement(now, "b", Lithium, 1, 1, "g"))
	a, _ = a.Append(measurement(now, "a", Nickel, 2, 2, "g"))
	b, _ := NewReport("b", "t", now)
	b, _ = b.Append(measurement(now, "a", Nickel, 2, 2, "g"))
	b, _ = b.Append(measurement(now, "b", Lithium, 1, 1, "g"))
	if CanonicalDigest(a) != CanonicalDigest(b) {
		t.Fatal("digest depends on insertion order")
	}
}
func TestSummarizeNormalizesUnits(t *testing.T) {
	now := time.Now()
	r, _ := NewReport("r", "t", now)
	r, _ = r.Append(measurement(now, "a", Lithium, 2, 2, "kg"))
	r, _ = r.Append(measurement(now, "b", Lithium, 100, 90, "g"))
	if got := Summarize(r)[Lithium]; got != 2090 {
		t.Fatalf("summary=%d", got)
	}
}
