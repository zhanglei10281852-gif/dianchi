package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
)

func TestRejectedPolicyUpdateDoesNotBecomeEffective(t *testing.T) {
	ctx := context.Background()
	operator := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	supervisor := auth.Principal{ID: "supervisor", TenantID: "plant-a", Role: auth.Supervisor}
	svc := NewPolicyService()
	unauthorized := Policy{MaxHazard: 0, ExpiryGrace: time.Hour, AllowedChemistries: map[string]bool{"LFP": true}}
	if err := svc.Set(ctx, operator, unauthorized); err == nil {
		t.Fatal("operator changed tenant policy")
	}
	lot := battery.Lot{ID: "lot-11", Chemistry: "LFP", HazardScore: 10}
	if err := svc.Allow(operator.TenantID, lot, time.Now()); err != nil {
		t.Fatalf("rejected policy became effective: %v", err)
	}
	approved := Policy{MaxHazard: 5, ExpiryGrace: time.Hour, AllowedChemistries: map[string]bool{"LFP": true}}
	if err := svc.Set(ctx, supervisor, approved); err != nil {
		t.Fatal(err)
	}
	if err := svc.Allow(operator.TenantID, lot, time.Now()); err == nil {
		t.Fatal("approved hazard limit was not enforced")
	}
}
