package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/safety"
)

func TestCanceledSafetyResolutionKeepsFindingOpen(t *testing.T) {
	ctx := context.Background()
	inspector := auth.Principal{ID: "inspector", TenantID: "plant-a", Role: auth.Operator}
	supervisor := auth.Principal{ID: "supervisor", TenantID: "plant-a", Role: auth.Supervisor}
	svc := NewSafetyService()
	finding := safety.Finding{ID: "finding-9", LotID: "lot-9", Code: "temperature", Measured: 90, Limit: 50}
	if err := svc.Record(ctx, inspector, finding); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := svc.Resolve(canceled, supervisor, finding.LotID, finding.ID); err == nil {
		t.Fatal("canceled resolution succeeded")
	}
	if open := svc.Open(supervisor, finding.LotID); len(open) != 1 {
		t.Fatalf("canceled resolution removed open finding: %+v", open)
	}
	if err := svc.Resolve(ctx, supervisor, finding.LotID, finding.ID); err != nil {
		t.Fatalf("active supervisor could not resolve finding: %v", err)
	}
}
