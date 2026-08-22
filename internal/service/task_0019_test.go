package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/material"
)

func TestCanceledMaterialAddDoesNotChangeInventory(t *testing.T) {
	base := context.Background()
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	svc := NewMaterialService()
	lot := material.Lot{ID: "material-19", Kind: material.Lithium, Grams: 200, Grade: material.GradeA, AssayedAt: time.Now()}
	canceled, cancel := context.WithCancel(base)
	cancel()
	if err := svc.Add(canceled, p, lot); err == nil {
		t.Fatal("canceled material add succeeded")
	}
	if rows := svc.List(base, p.TenantID, material.Lithium); len(rows) != 0 {
		t.Fatalf("canceled add changed inventory: %+v", rows)
	}
	lot.ID = "material-19-live"
	if err := svc.Add(base, p, lot); err != nil {
		t.Fatalf("live material add failed: %v", err)
	}
}
