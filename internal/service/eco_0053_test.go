package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"testing"
)

func TestEcoPolicyCopiesChemistryRules(t *testing.T) {
	s := NewPolicyService()
	p := auth.Principal{ID: "s", TenantID: "e", Role: auth.Supervisor}
	rules := map[string]bool{"LFP": true}
	if e := s.Set(context.Background(), p, Policy{MaxHazard: 50, AllowedChemistries: rules}); e != nil {
		t.Fatal(e)
	}
	rules["LFP"] = false
	if !s.Get("e").AllowedChemistries["LFP"] {
		t.Fatal("map alias")
	}
}
