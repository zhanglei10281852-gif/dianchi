package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/manifest"
	"sync"
	"time"
)

type ManifestService struct {
	mu   sync.RWMutex
	data map[string]manifest.Manifest
}

func NewManifestService() *ManifestService {
	return &ManifestService{data: map[string]manifest.Manifest{}}
}
func (m *ManifestService) Create(ctx context.Context, p auth.Principal, v manifest.Manifest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("recover") {
		return apperr.New(apperr.Forbidden, "role cannot create manifest")
	}
	if err := manifest.Validate(v); err != nil {
		return apperr.Wrap(apperr.Invalid, "manifest", err)
	}
	v.Status = manifest.Prepared
	v.CreatedBy = p.ID
	v.CreatedAt = time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[v.ID]; ok {
		return apperr.New(apperr.Conflict, "manifest exists")
	}
	m.data[v.ID] = v
	return nil
}
func (m *ManifestService) AddItem(ctx context.Context, p auth.Principal, id string, item manifest.Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[id]
	if !ok || v.TenantID != p.TenantID {
		return apperr.New(apperr.NotFound, "manifest missing")
	}
	if v.Status != manifest.Prepared {
		return apperr.New(apperr.Conflict, "manifest sealed")
	}
	item.ManifestID = id
	item.TenantID = p.TenantID
	v.Items = append(v.Items, item)
	m.data[id] = v
	return nil
}
func (m *ManifestService) Move(ctx context.Context, p auth.Principal, id string, target manifest.Status) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[id]
	if !ok || v.TenantID != p.TenantID {
		return apperr.New(apperr.NotFound, "manifest missing")
	}
	if err := manifest.Transition(v.Status, target); err != nil {
		return apperr.Wrap(apperr.Conflict, "manifest transition", err)
	}
	v.Status = target
	now := time.Now().UTC()
	if target == manifest.Sealed {
		v.SealedAt = &now
	}
	if target == manifest.Received {
		v.ReceivedAt = &now
	}
	m.data[id] = v
	return nil
}
func (m *ManifestService) Get(ctx context.Context, p auth.Principal, id string) (manifest.Manifest, error) {
	if err := ctx.Err(); err != nil {
		return manifest.Manifest{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[id]
	if !ok || v.TenantID != p.TenantID {
		return manifest.Manifest{}, apperr.New(apperr.NotFound, "manifest missing")
	}
	v.Items = append([]manifest.Item(nil), v.Items...)
	return v, nil
}
func (m *ManifestService) Pending(tenant string) []manifest.Manifest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]manifest.Manifest, 0)
	for _, v := range m.data {
		if v.TenantID == tenant && v.Status != manifest.Received && v.Status != manifest.Cancelled {
			out = append(out, v)
		}
	}
	return out
}
