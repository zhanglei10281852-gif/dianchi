package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/compliance"
	"testing"
)

func TestEcoComplianceInvalidSubmitStaysDraft(t *testing.T) {
	s := NewComplianceService()
	p := auth.Principal{ID: "s", TenantID: "e", Role: auth.Supervisor}
	c, e := s.Draft(context.Background(), p, "lot", "serial", []string{"mass"})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Submit(context.Background(), p, c.ID, map[string]string{"mass": "ok"}); e == nil {
		t.Fatal("accepted")
	}
	got, e := s.Get(context.Background(), p, c.ID)
	if e != nil {
		t.Fatal(e)
	}
	if got.Status != compliance.Draft {
		t.Fatalf("status=%s", got.Status)
	}
}
