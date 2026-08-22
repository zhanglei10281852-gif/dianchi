package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/compliance"
)

func TestRejectedComplianceReviewLeavesCertificateUnchanged(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "supervisor", TenantID: "plant-a", Role: auth.Supervisor}
	svc := NewComplianceService()
	cert, err := svc.Draft(ctx, p, "lot-20", "CERT-20", []string{"origin"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Review(ctx, p, cert.ID, true, "too early"); err == nil {
		t.Fatal("draft certificate was reviewed")
	}
	got, err := svc.Get(ctx, p, cert.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != compliance.Draft {
		t.Fatalf("failed review changed certificate to %s", got.Status)
	}
	evidence := map[string]string{"mass-balance": "ok", "safety": "ok", "origin": "ok"}
	if err := svc.Submit(ctx, p, cert.ID, evidence); err != nil {
		t.Fatal(err)
	}
	if err := svc.Review(ctx, p, cert.ID, true, "verified"); err != nil {
		t.Fatalf("legal review failed: %v", err)
	}
}
