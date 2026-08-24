package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"sync"
	"time"
)

type Policy struct {
	MaxHazard          int
	RequireInspection  bool
	ExpiryGrace        time.Duration
	AllowedChemistries map[string]bool
}
type PolicyService struct {
	mu       sync.RWMutex
	policies map[string]Policy
}

func NewPolicyService() *PolicyService { return &PolicyService{policies: map[string]Policy{}} }
func (p *PolicyService) Set(ctx context.Context, principal auth.Principal, v Policy) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if principal.Role != auth.Supervisor {
		return apperr.New(apperr.Forbidden, "supervisor required")
	}
	if v.MaxHazard < 0 || v.MaxHazard > 100 {
		return apperr.Invalidf("hazard limit invalid")
	}
	if v.ExpiryGrace < 0 {
		return apperr.Invalidf("expiry grace invalid")
	}
	candidate := v
	candidate.AllowedChemistries = copyMap(v.AllowedChemistries)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.policies[principal.TenantID] = candidate
	return nil
}
func (p *PolicyService) Get(tenant string) Policy {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.policies[tenant]
	if !ok {
		return Policy{MaxHazard: 70, RequireInspection: true, ExpiryGrace: 24 * time.Hour, AllowedChemistries: map[string]bool{"LFP": true, "NMC": true, "LMO": true}}
	}
	v.AllowedChemistries = copyMap(v.AllowedChemistries)
	return v
}
func (p *PolicyService) Allow(tenant string, lot battery.Lot, now time.Time) error {
	v := p.Get(tenant)
	if err := lot.ValidatePolicy(v.MaxHazard, v.AllowedChemistries, v.ExpiryGrace, now); err != nil {
		return apperr.Wrap(apperr.Conflict, "lot policy", err)
	}
	return nil
}
func copyMap(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
