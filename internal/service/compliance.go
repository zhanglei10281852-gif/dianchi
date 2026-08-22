package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/compliance"
)

type ComplianceService struct {
	mu           sync.RWMutex
	certs        map[string]compliance.Certificate
	reviews      map[string][]compliance.Review
	requirements []compliance.Requirement
}

func NewComplianceService() *ComplianceService {
	return &ComplianceService{certs: map[string]compliance.Certificate{}, reviews: map[string][]compliance.Review{}, requirements: []compliance.Requirement{{Code: "mass-balance", Required: true}, {Code: "safety", Required: true}, {Code: "origin", Required: true}}}
}
func (c *ComplianceService) Draft(ctx context.Context, p auth.Principal, lotID, serial string, evidence []string) (compliance.Certificate, error) {
	if err := ctx.Err(); err != nil {
		return compliance.Certificate{}, err
	}
	if !p.Can("certify") {
		return compliance.Certificate{}, apperr.New(apperr.Forbidden, "only supervisors create certificates")
	}
	now := time.Now().UTC()
	out := compliance.Certificate{ID: token(), TenantID: p.TenantID, LotID: lotID, Serial: serial, Issuer: p.ID, Status: compliance.Draft, IssuedAt: now, ExpiresAt: now.AddDate(1, 0, 0), Evidence: compliance.MergeEvidence(nil, evidence)}
	if err := compliance.Validate(out); err != nil {
		return out, apperr.Wrap(apperr.Invalid, "certificate", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.certs[out.ID]; ok {
		return out, apperr.New(apperr.Conflict, "certificate exists")
	}
	c.certs[out.ID] = out
	return out, nil
}
func (c *ComplianceService) Submit(ctx context.Context, p auth.Principal, id string, provided map[string]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	cert, ok := c.certs[id]
	if !ok || cert.TenantID != p.TenantID {
		return apperr.New(apperr.NotFound, "certificate missing")
	}
	candidate := cert
	candidate.Status = compliance.Submitted
	c.certs[id] = candidate
	if err := compliance.ValidateSubmission(cert, c.requirements, provided); err != nil {
		return apperr.Wrap(apperr.Invalid, "certificate submission", err)
	}
	return nil
}
func (c *ComplianceService) Review(ctx context.Context, p auth.Principal, id string, approve bool, notes string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("certify") {
		return apperr.New(apperr.Forbidden, "only supervisors review")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	cert, ok := c.certs[id]
	if !ok || cert.TenantID != p.TenantID {
		return apperr.New(apperr.NotFound, "certificate missing")
	}
	target := compliance.Rejected
	if approve {
		target = compliance.Approved
	}
	if err := compliance.Transition(cert.Status, target); err != nil {
		return apperr.Wrap(apperr.Conflict, "review transition", err)
	}
	cert.Status = target
	c.certs[id] = cert
	c.reviews[id] = append(c.reviews[id], compliance.Review{ID: token(), CertificateID: id, ReviewerID: p.ID, Decision: string(target), Notes: notes, ReviewedAt: time.Now().UTC()})
	return nil
}
func (c *ComplianceService) Get(ctx context.Context, p auth.Principal, id string) (compliance.Certificate, error) {
	if err := ctx.Err(); err != nil {
		return compliance.Certificate{}, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.certs[id]
	if !ok || v.TenantID != p.TenantID {
		return compliance.Certificate{}, apperr.New(apperr.NotFound, "certificate missing")
	}
	v.Evidence = append([]string(nil), v.Evidence...)
	return v, nil
}
func (c *ComplianceService) Summary(tenant string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	counts := map[compliance.Status]int{}
	for _, v := range c.certs {
		if v.TenantID == tenant {
			counts[v.Status]++
		}
	}
	return fmt.Sprintf("draft=%d submitted=%d approved=%d rejected=%d", counts[compliance.Draft], counts[compliance.Submitted], counts[compliance.Approved], counts[compliance.Rejected])
}
