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
	if err := safety.Validate(f); err != nil {
		return apperr.Wrap(apperr.Invalid, "finding", err)
	}
	f.TenantID = p.TenantID
	f.ObservedAt = time.Now().UTC()
	f.Level = safety.Assess(f.Measured, f.Limit)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.findings[f.LotID] = append(s.findings[f.LotID], f)
	return nil
}
func (s *SafetyService) Resolve(ctx context.Context, p auth.Principal, lotID, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("certify") {
		return apperr.New(apperr.Forbidden, "supervisor required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, f := range s.findings[lotID] {
		if f.ID == id && f.TenantID == p.TenantID {
			s.findings[lotID][i] = safety.Resolve(f)
			return nil
		}
	}
	return apperr.New(apperr.NotFound, "finding missing")
}
func (s *SafetyService) Open(lotID string) []safety.Finding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]safety.Finding, 0)
	for _, f := range s.findings[lotID] {
		if f.Open() {
			out = append(out, f)
		}
	}
	return out
}
func (s *SafetyService) Checklist(lotID string) safety.Checklist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return safety.Checklist{ID: token(), LotID: lotID, Findings: append([]safety.Finding(nil), s.findings[lotID]...)}
}
