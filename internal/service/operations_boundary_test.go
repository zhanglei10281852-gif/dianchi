package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/energy"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/manifest"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/material"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/operator"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/safety"
)

func boundaryPrincipals() (auth.Principal, auth.Principal, auth.Principal) {
	return auth.Principal{ID: "op-a", TenantID: "tenant-a", Role: auth.Operator},
		auth.Principal{ID: "op-b", TenantID: "tenant-b", Role: auth.Operator},
		auth.Principal{ID: "sup-a", TenantID: "tenant-a", Role: auth.Supervisor}
}

func TestMaterialReservationIsTenantScopedAndConsumesInventory(t *testing.T) {
	opA, opB, _ := boundaryPrincipals()
	svc := NewMaterialService()
	lot := material.Lot{ID: "m-1", TenantID: "tenant-b", Kind: material.Lithium, Grams: 100, Grade: material.GradeA, AssayedAt: time.Now()}
	if err := svc.Add(context.Background(), opA, lot); err != nil {
		t.Fatal(err)
	}
	if got := svc.List(context.Background(), "tenant-b", material.Lithium); len(got) != 0 {
		t.Fatalf("caller supplied tenant was trusted: %+v", got)
	}
	selected, err := svc.Reserve(context.Background(), opA, "tenant-a", material.Lithium, 60)
	if err != nil || len(selected) != 1 || selected[0].Grams != 60 {
		t.Fatalf("reservation=%+v err=%v", selected, err)
	}
	if _, err := svc.Reserve(context.Background(), opA, "tenant-a", material.Lithium, 50); err == nil {
		t.Fatal("consumed inventory was available again")
	}
	if _, err := svc.Reserve(context.Background(), opB, "tenant-a", material.Lithium, 1); err == nil {
		t.Fatal("other tenant reserved inventory")
	}
}

func TestMaterialFailedReservationDoesNotConsumeInventory(t *testing.T) {
	op, _, _ := boundaryPrincipals()
	svc := NewMaterialService()
	for _, lot := range []material.Lot{
		{ID: "m-1", Kind: material.Nickel, Grams: 20, Grade: material.GradeA, AssayedAt: time.Now()},
		{ID: "m-2", Kind: material.Nickel, Grams: 30, Grade: material.GradeB, AssayedAt: time.Now()},
	} {
		if err := svc.Add(context.Background(), op, lot); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Reserve(context.Background(), op, op.TenantID, material.Nickel, 60); err == nil {
		t.Fatal("oversized reservation succeeded")
	}
	selected, err := svc.Reserve(context.Background(), op, op.TenantID, material.Nickel, 50)
	if err != nil || len(selected) != 2 {
		t.Fatalf("rollback reservation=%+v err=%v", selected, err)
	}
}

func TestEnergyReadingsWithSameLotIDStayInTenant(t *testing.T) {
	opA, opB, _ := boundaryPrincipals()
	svc := NewEnergyService()
	for _, input := range []struct {
		principal auth.Principal
		voltage   float64
	}{
		{opA, 10},
		{opB, 20},
	} {
		r := energy.Reading{ID: input.principal.ID, TenantID: "forged", LotID: "shared-lot", Voltage: input.voltage, Current: 2, Temperature: 20, At: time.Now()}
		if err := svc.Record(context.Background(), input.principal, r); err != nil {
			t.Fatal(err)
		}
	}
	avgA, err := svc.Average(context.Background(), opA, "shared-lot")
	if err != nil || avgA != 20 {
		t.Fatalf("tenant A average=%v err=%v", avgA, err)
	}
	avgB, err := svc.Average(context.Background(), opB, "shared-lot")
	if err != nil || avgB != 40 {
		t.Fatalf("tenant B average=%v err=%v", avgB, err)
	}
}

func TestCancelledRecordExcludesReadingFromQueries(t *testing.T) {
	op, _, _ := boundaryPrincipals()
	svc := NewEnergyService()

	base := energy.Reading{ID: "keep", LotID: "lot", Voltage: 10, Current: 2, Temperature: 20, At: time.Unix(1, 0)}
	if err := svc.Record(context.Background(), op, base); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.Record(ctx, op, energy.Reading{ID: "dropped", LotID: "lot", Voltage: 99, Current: 1, Temperature: 20, At: time.Unix(2, 0)}); err == nil {
		t.Fatal("cancelled record succeeded")
	}

	avg, err := svc.Average(context.Background(), op, "lot")
	if err != nil || avg != 20 {
		t.Fatalf("average=%v err=%v want 20", avg, err)
	}
	rows := svc.Between(context.Background(), op, "lot", time.Unix(0, 0), time.Unix(3, 0))
	if len(rows) != 1 || rows[0].ID != "keep" {
		t.Fatalf("between=%+v", rows)
	}
}

func TestManifestAndOperatorInputsAreCopied(t *testing.T) {
	op, _, sup := boundaryPrincipals()
	manifests := NewManifestService()
	v := manifest.Manifest{ID: "manifest", TenantID: "forged", Origin: "A", Destination: "B"}
	if err := manifests.Create(context.Background(), op, v); err != nil {
		t.Fatal(err)
	}
	got, err := manifests.Get(context.Background(), op, v.ID)
	if err != nil || got.TenantID != op.TenantID {
		t.Fatalf("manifest=%+v err=%v", got, err)
	}
	operators := NewOperatorService()
	certs := map[operator.Certification]time.Time{operator.Hazmat: time.Now().Add(time.Hour)}
	worker := operator.Operator{ID: "worker", Name: "Worker", Shift: operator.Day, Active: true, Certifications: certs}
	if err := operators.Put(context.Background(), sup, worker); err != nil {
		t.Fatal(err)
	}
	certs[operator.Hazmat] = time.Time{}
	stored, err := operators.Get(context.Background(), sup, worker.ID)
	if err != nil || stored.Certifications[operator.Hazmat].IsZero() {
		t.Fatalf("operator input mutation leaked: %+v err=%v", stored, err)
	}
}

func TestSafetyAndQuarantineUseTenantQualifiedKeys(t *testing.T) {
	opA, opB, supA := boundaryPrincipals()
	svc := NewSafetyService()
	finding := safety.Finding{ID: "same", LotID: "shared", Code: "heat", Description: "temperature", Measured: 80, Limit: 60}
	if err := svc.Record(context.Background(), opA, finding); err != nil {
		t.Fatal(err)
	}
	if err := svc.Record(context.Background(), opB, finding); err != nil {
		t.Fatal(err)
	}
	if err := svc.Resolve(context.Background(), supA, "shared", "same"); err != nil {
		t.Fatal(err)
	}
	if len(svc.Open(opA, "shared")) != 0 || len(svc.Open(opB, "shared")) != 1 {
		t.Fatal("safety resolution crossed tenant boundary")
	}
	quarantine := NewQuarantine()
	lot := battery.Lot{ID: "shared", TenantID: "forged", State: battery.Quarantined, ReceivedAt: time.Now()}
	if err := quarantine.Hold(context.Background(), opA, lot, "tenant A"); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := quarantine.Get(opB, lot.ID); ok {
		t.Fatal("quarantine record leaked to tenant B")
	}
}
