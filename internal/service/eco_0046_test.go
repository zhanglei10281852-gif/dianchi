package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"testing"
)

func TestEcoComplianceEvidenceSnapshot(t *testing.T) {
	s := NewComplianceService()
	p := auth.Principal{ID: "s", TenantID: "e", Role: auth.Supervisor}
	c, e := s.Draft(context.Background(), p, "lot", "serial", []string{"evidence"})
	if e != nil {
		t.Fatal(e)
	}
	got, e := s.Get(context.Background(), p, c.ID)
	if e != nil {
		t.Fatal(e)
	}
	got.Evidence[0] = "tampered"
	fresh, _ := s.Get(context.Background(), p, c.ID)
	if fresh.Evidence[0] != "evidence" {
		t.Fatalf("%v", fresh.Evidence)
	}
}
