package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/operator"
	"sync"
	"time"
)

type OperatorService struct {
	mu    sync.RWMutex
	items map[string]operator.Operator
}

func NewOperatorService() *OperatorService {
	return &OperatorService{items: map[string]operator.Operator{}}
}
func (o *OperatorService) Put(ctx context.Context, p auth.Principal, v operator.Operator) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.Role != auth.Supervisor {
		return apperr.New(apperr.Forbidden, "supervisor required")
	}
	v.TenantID = p.TenantID
	if err := operator.Validate(v); err != nil {
		return apperr.Wrap(apperr.Invalid, "operator", err)
	}
	v.Certifications = operator.MergeCertifications(nil, v.Certifications)
	o.mu.Lock()
	defer o.mu.Unlock()
	o.items[v.ID] = v
	return nil
}
func (o *OperatorService) Grant(ctx context.Context, p auth.Principal, id string, cert operator.Certification, expires time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.Role != auth.Supervisor {
		return apperr.New(apperr.Forbidden, "supervisor required")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	v, ok := o.items[id]
	if !ok || v.TenantID != p.TenantID {
		return apperr.New(apperr.NotFound, "operator missing")
	}
	if v.Certifications == nil {
		v.Certifications = map[operator.Certification]time.Time{}
	}
	v.Certifications[cert] = expires
	o.items[id] = v
	return nil
}
func (o *OperatorService) Get(ctx context.Context, p auth.Principal, id string) (operator.Operator, error) {
	if err := ctx.Err(); err != nil {
		return operator.Operator{}, err
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	v, ok := o.items[id]
	if !ok || v.TenantID != p.TenantID {
		return operator.Operator{}, apperr.New(apperr.NotFound, "operator missing")
	}
	v.Certifications = operator.MergeCertifications(nil, v.Certifications)
	return v, nil
}
