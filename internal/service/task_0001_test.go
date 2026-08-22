package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/compliance"
)

func TestRejectedComplianceSubmissionPreservesDraft(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "supervisor", TenantID: "plant-a", Role: auth.Supervisor}
	svc := NewComplianceService()
	cert, err := svc.Draft(ctx, p, "lot-17", "CERT-17", []string{"lab-result"})
	if err != nil {
		t.Fatal(err)
	}
	err = svc.Submit(ctx, p, cert.ID, map[string]string{"mass-balance": "ok"})
	if err == nil {
		t.Fatal("submission without all mandatory evidence succeeded")
	}
	got, err := svc.Get(ctx, p, cert.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != compliance.Draft {
		t.Fatalf("rejected submission changed status to %s", got.Status)
	}
	complete := map[string]string{"mass-balance": "ok", "safety": "ok", "origin": "ok"}
	if err := svc.Submit(ctx, p, cert.ID, complete); err != nil {
		t.Fatalf("corrected submission failed: %v", err)
	}
}
