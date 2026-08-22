package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
)

func TestCanceledAnalyticsRecordDoesNotReachDailySeries(t *testing.T) {
	base := context.Background()
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	svc := &Analytics{}
	at := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	canceled, cancel := context.WithCancel(base)
	cancel()
	if err := svc.Record(canceled, p, battery.Recovered, at); err == nil {
		t.Fatal("canceled analytics record succeeded")
	}
	if rows := svc.Daily(base, p.TenantID, at.Add(-time.Hour), at.Add(time.Hour)); len(rows) != 0 {
		t.Fatalf("canceled event reached daily analytics: %+v", rows)
	}
	if err := svc.Record(base, p, battery.Recovered, at); err != nil {
		t.Fatalf("live analytics record failed: %v", err)
	}
}
