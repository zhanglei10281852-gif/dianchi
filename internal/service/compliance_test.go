package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/compliance"
)

func errCode(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestComplianceReviewConflictKeepsStatus(t *testing.T) {
	svc := NewComplianceService()
	sup := auth.Principal{ID: "sup", TenantID: "t", Role: auth.Supervisor}
	ctx := context.Background()

	cert, err := svc.Draft(ctx, sup, "lot-1", "SN-1", []string{"evidence-1"})
	if err != nil {
		t.Fatal(err)
	}

	// Supervisor misclicks approve on a not-yet-submitted draft certificate.
	if err := svc.Review(ctx, sup, cert.ID, true, "misclick"); err == nil {
		t.Fatal("expected conflict reviewing draft certificate")
	}

	got, err := svc.Get(ctx, sup, cert.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != compliance.Draft {
		t.Fatalf("rejected review rewrote status: got %s want %s", got.Status, compliance.Draft)
	}

	// The original draft must remain submittable.
	if err := svc.Submit(ctx, sup, cert.ID, map[string]string{
		"mass-balance": "ok", "safety": "ok", "origin": "ok",
	}); err != nil {
		t.Fatalf("submit blocked after rejected review: %v", err)
	}

	// A legitimate approve now succeeds on the submitted certificate.
	if err := svc.Review(ctx, sup, cert.ID, true, "approved"); err != nil {
		t.Fatalf("approve after submit failed: %v", err)
	}
	got, _ = svc.Get(ctx, sup, cert.ID)
	if got.Status != compliance.Approved {
		t.Fatalf("status=%s want %s", got.Status, compliance.Approved)
	}

	// The misclick review must not have been recorded.
	if n := len(svc.reviews[cert.ID]); n != 1 {
		t.Fatalf("recorded %d reviews, want 1", n)
	}
}

func TestComplianceReviewUnknownCertificate(t *testing.T) {
	svc := NewComplianceService()
	sup := auth.Principal{ID: "sup", TenantID: "t", Role: auth.Supervisor}
	err := svc.Review(context.Background(), sup, "missing", true, "")
	if err == nil {
		t.Fatal("expected error for missing certificate")
	}
	if code := errCode(err); code != apperr.NotFound {
		t.Fatalf("code=%v want %v", code, apperr.NotFound)
	}
}
