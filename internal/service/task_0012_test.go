package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/operator"
)

func TestUnauthorizedCertificationGrantLeavesOperatorUntouched(t *testing.T) {
	ctx := context.Background()
	supervisor := auth.Principal{ID: "supervisor", TenantID: "plant-a", Role: auth.Supervisor}
	worker := auth.Principal{ID: "worker", TenantID: "plant-a", Role: auth.Operator}
	svc := NewOperatorService()
	profile := operator.Operator{ID: "worker-12", Name: "Lin", Shift: operator.Day, Active: true}
	if err := svc.Put(ctx, supervisor, profile); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(24 * time.Hour)
	if err := svc.Grant(ctx, worker, profile.ID, operator.Hazmat, expires); err == nil {
		t.Fatal("operator granted hazardous-material certificate")
	}
	got, err := svc.Get(ctx, supervisor, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := got.Certifications[operator.Hazmat]; exists {
		t.Fatalf("unauthorized grant remained on operator: %+v", got.Certifications)
	}
	if err := svc.Grant(ctx, supervisor, profile.ID, operator.Hazmat, expires); err != nil {
		t.Fatalf("supervisor grant failed: %v", err)
	}
}
