package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/material"
	"strings"
	"testing"
	"time"
)

func TestEcoAssaySummaryIsTenantScoped(t *testing.T) {
	s := NewMaterialService()
	a := auth.Principal{ID: "a", TenantID: "a", Role: auth.Operator}
	b := auth.Principal{ID: "b", TenantID: "b", Role: auth.Operator}
	for _, x := range []struct {
		p  auth.Principal
		id string
	}{{a, "a1"}, {b, "b1"}} {
		if e := s.AddAssay(context.Background(), x.p, material.Assay{ID: x.id, RecoveryID: x.id, Kind: material.Lithium, PurityPPM: 990000, MoisturePPM: 50, AssayedAt: time.Now()}); e != nil {
			t.Fatal(e)
		}
	}
	if got := s.AssaySummary("a"); !strings.HasPrefix(got, "1/1") {
		t.Fatalf("summary=%s", got)
	}
}
