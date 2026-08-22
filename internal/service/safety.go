package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/safety"
	"sync"
	"time"
)

type SafetyService struct {
	mu       sync.RWMutex
	findings map[string][]safety.Finding
}

func NewSafetyService() *SafetyService {
	return &SafetyService{findings: map[string][]safety.Finding{}}
}
func (s *SafetyService) Record(ctx context.Context, p auth.Principal, f safety.Finding) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("inspect") {
		return apperr.New(apperr.Forbidden, "role cannot record finding")
	}
	f.TenantID = p.TenantID
	if err := safety.Validate(f); err != nil {
		return apperr.Wrap(apperr.Invalid, "finding", err)
	}
	f.ObservedAt = time.Now().UTC()
	f.Level = safety.Assess(f.Measured, f.Limit)
	s.mu.Lock()
	defer s.mu.Unlock()
	key := safetyKey(p.TenantID, f.LotID)
	s.findings[key] = append(s.findings[key], f)
	return nil
}
func (s *SafetyService) Resolve(ctx context.Context, p auth.Principal, lotID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := safetyKey(p.TenantID, lotID)
	for i, f := range s.findings[key] {
		if f.ID != id || f.TenantID != p.TenantID {
			continue
		}
		resolved, err := safety.Resolve(f)
		if err != nil {
			return apperr.Wrap(apperr.Conflict, "finding resolution", err)
		}
		s.findings[key][i] = resolved
		if err := ctx.Err(); err != nil {
			return err
		}
		if !p.Can("certify") {
			return apperr.New(apperr.Forbidden, "supervisor required")
		}
		return nil
	}
	return apperr.New(apperr.NotFound, "finding missing")
}
func (s *SafetyService) Open(p auth.Principal, lotID string) []safety.Finding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]safety.Finding, 0)
	for _, f := range s.findings[safetyKey(p.TenantID, lotID)] {
		if f.Open() {
			out = append(out, f)
		}
	}
	return out
}
func (s *SafetyService) Checklist(p auth.Principal, lotID string) safety.Checklist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return safety.Checklist{ID: token(), LotID: lotID, InspectorID: p.ID, Findings: append([]safety.Finding(nil), s.findings[safetyKey(p.TenantID, lotID)]...)}
}

func safetyKey(tenant, lot string) string { return tenant + "\x00" + lot }
