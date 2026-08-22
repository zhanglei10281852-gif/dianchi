package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"testing"
)

func TestEcoComplianceRejectedReviewDoesNotLeaveRecord(t *testing.T) {
	s := NewComplianceService()
	p := auth.Principal{ID: "s", TenantID: "e", Role: auth.Supervisor}
	c, e := s.Draft(context.Background(), p, "lot", "serial", []string{"x"})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Review(context.Background(), p, c.ID, false, "bad"); e == nil {
		t.Fatal("accepted")
	}
	s.mu.RLock()
	n := len(s.reviews[c.ID])
	s.mu.RUnlock()
	if n != 0 {
		t.Fatal(n)
	}
}
