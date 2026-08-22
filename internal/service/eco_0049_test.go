package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/incident"
	"testing"
)

func TestEcoIncidentResolutionRecordsClosure(t *testing.T) {
	s := NewIncidentService()
	p := auth.Principal{ID: "o", TenantID: "e", Role: auth.Operator}
	if _, e := s.Open(context.Background(), p, incident.Incident{ID: "i", LotID: "l", Summary: "leak", Severity: incident.High}); e != nil {
		t.Fatal(e)
	}
	if e := s.Move(context.Background(), p, "i", incident.Investigating); e != nil {
		t.Fatal(e)
	}
	if e := s.Move(context.Background(), p, "i", incident.Contained); e != nil {
		t.Fatal(e)
	}
	if e := s.Move(context.Background(), p, "i", incident.Resolved); e != nil {
		t.Fatal(e)
	}
	got, _, e := s.Get(context.Background(), p, "i")
	if e != nil {
		t.Fatal(e)
	}
	if got.ClosedAt == nil {
		t.Fatal("missing closure")
	}
}
