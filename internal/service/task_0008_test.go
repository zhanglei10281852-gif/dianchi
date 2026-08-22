package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/incident"
)

func TestRejectedIncidentActionDoesNotAdvanceInvestigation(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "inspector", TenantID: "plant-a", Role: auth.Operator}
	svc := NewIncidentService()
	created, err := svc.Open(ctx, p, incident.Incident{ID: "incident-8", LotID: "lot-8", Summary: "damaged casing", Severity: incident.Medium})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddAction(ctx, p, created.ID, " "); err == nil {
		t.Fatal("blank corrective action was accepted")
	}
	got, actions, err := svc.Get(ctx, p, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != incident.Open || len(actions) != 0 {
		t.Fatalf("rejected action changed incident: status=%s actions=%d", got.Status, len(actions))
	}
	if err := svc.AddAction(ctx, p, created.ID, "isolate container"); err != nil {
		t.Fatalf("valid corrective action failed: %v", err)
	}
}
