package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/material"
)

func TestInvalidAssayDoesNotEnterMaterialSummary(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "inspector", TenantID: "plant-a", Role: auth.Operator}
	svc := NewMaterialService()
	invalid := material.Assay{ID: "assay-20", RecoveryID: "recovery-20", Kind: material.Lithium, PurityPPM: -1, MoisturePPM: 20, AssayedAt: time.Now()}
	if err := svc.AddAssay(ctx, p, invalid); err == nil {
		t.Fatal("invalid assay was accepted")
	}
	if summary := svc.AssaySummary(p.TenantID); summary != "0/0 assays pass" {
		t.Fatalf("invalid assay polluted summary: %s", summary)
	}
	valid := invalid
	valid.ID = "assay-20-live"
	valid.PurityPPM = 995000
	if err := svc.AddAssay(ctx, p, valid); err != nil {
		t.Fatalf("valid assay failed: %v", err)
	}
}
