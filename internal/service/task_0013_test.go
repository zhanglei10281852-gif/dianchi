package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/energy"
)

func TestCanceledEnergyRecordDoesNotAffectReadings(t *testing.T) {
	base := context.Background()
	p := auth.Principal{ID: "inspector", TenantID: "plant-a", Role: auth.Operator}
	svc := NewEnergyService()
	now := time.Now().UTC()
	reading := energy.Reading{ID: "reading-13", LotID: "lot-13", Voltage: 10, Current: 2, Temperature: 25, At: now}
	canceled, cancel := context.WithCancel(base)
	cancel()
	if err := svc.Record(canceled, p, reading); err == nil {
		t.Fatal("canceled energy record succeeded")
	}
	if rows := svc.Between(base, p, reading.LotID, now.Add(-time.Minute), now.Add(time.Minute)); len(rows) != 0 {
		t.Fatalf("canceled reading remained visible: %+v", rows)
	}
	reading.ID = "reading-13-live"
	if err := svc.Record(base, p, reading); err != nil {
		t.Fatalf("live energy record failed: %v", err)
	}
}
