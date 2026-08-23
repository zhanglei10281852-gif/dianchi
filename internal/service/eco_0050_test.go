package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/safety"
	"testing"
)

func TestEcoChecklistSnapshot(t *testing.T) {
	s := NewSafetyService()
	p := auth.Principal{ID: "o", TenantID: "e", Role: auth.Operator}
	if e := s.Record(context.Background(), p, safety.Finding{ID: "f", LotID: "l", Code: "heat", Description: "temperature", Measured: 90, Limit: 60}); e != nil {
		t.Fatal(e)
	}
	c := s.Checklist(p, "l")
	c.Findings[0].Description = "tampered"
	got := s.Open(p, "l")
	if len(got) != 1 || got[0].Description != "temperature" {
		t.Fatalf("%+v", got)
	}
}
