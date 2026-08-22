package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/incident"
)

func TestRejectedIncidentResolutionKeepsIncidentOpen(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "inspector", TenantID: "plant-a", Role: auth.Operator}
	svc := NewIncidentService()
	created, err := svc.Open(ctx, p, incident.Incident{ID: "incident-7", LotID: "lot-7", Summary: "thermal alarm", Severity: incident.High})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Move(ctx, p, created.ID, incident.Resolved); err == nil {
		t.Fatal("open incident was resolved directly")
	}
	got, _, err := svc.Get(ctx, p, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != incident.Open || got.ClosedAt != nil {
		t.Fatalf("rejected resolution closed incident: %+v", got)
	}
	if err := svc.Move(ctx, p, created.ID, incident.Investigating); err != nil {
		t.Fatalf("investigation could not start: %v", err)
	}
}
