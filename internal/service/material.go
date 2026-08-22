package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/material"
	"github.com/zhanglei10281852-gif/dianchi/internal/validation"
)

type MaterialService struct {
	mu     sync.RWMutex
	lots   map[string][]material.Lot
	assays map[string][]material.Assay
}

func NewMaterialService() *MaterialService {
	return &MaterialService{lots: map[string][]material.Lot{}, assays: map[string][]material.Assay{}}
}
func (m *MaterialService) Add(ctx context.Context, p auth.Principal, l material.Lot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("recover") {
		return apperr.New(apperr.Forbidden, "role cannot add material")
	}
	l.TenantID = p.TenantID
	if err := validation.Tenant(l.TenantID); err != nil {
		return apperr.Wrap(apperr.Invalid, "tenant", err)
	}
	if err := material.ValidateLot(l); err != nil {
		return apperr.Wrap(apperr.Invalid, "material", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.lots[l.TenantID] {
		if v.ID == l.ID {
			return apperr.New(apperr.Conflict, "material already exists")
		}
	}
	m.lots[l.TenantID] = append(m.lots[l.TenantID], l)
	return nil
}
func (m *MaterialService) AddAssay(ctx context.Context, p auth.Principal, a material.Assay) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Can("inspect") {
		return apperr.New(apperr.Forbidden, "role cannot assay")
	}
	a.TenantID = p.TenantID
	if err := material.ValidateAssay(a); err != nil {
		return apperr.Wrap(apperr.Invalid, "assay", err)
	}
	a.Grade = material.Classify(a.PurityPPM, a.MoisturePPM)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assays[a.RecoveryID] = append(m.assays[a.RecoveryID], a)
	return nil
}
func (m *MaterialService) List(ctx context.Context, tenant string, kind material.Kind) []material.Lot {
	if ctx.Err() != nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]material.Lot, 0)
	for _, v := range m.lots[tenant] {
		if kind == "" || v.Kind == kind {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AssayedAt.Before(out[j].AssayedAt) })
	return out
}
func (m *MaterialService) Balance(ctx context.Context, tenant string) map[material.Kind]material.Balance {
	if ctx.Err() != nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[material.Kind]material.Balance{}
	for _, v := range m.lots[tenant] {
		b := out[v.Kind]
		b.Incoming += v.Grams
		if v.IsUsable() {
			b.Accepted += v.Grams
			b.Available += v.Grams
		} else {
			b.Rejected += v.Grams
		}
		out[v.Kind] = b
	}
	return out
}
func (m *MaterialService) Reserve(ctx context.Context, p auth.Principal, tenant string, kind material.Kind, grams int) ([]material.Lot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !p.Can("recover") {
		return nil, apperr.New(apperr.Forbidden, "role cannot reserve output")
	}
	if grams <= 0 {
		return nil, apperr.Invalidf("reserve amount must be positive")
	}
	if tenant != p.TenantID {
		return nil, apperr.New(apperr.Forbidden, "cannot reserve another tenant's material")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	remaining := grams
	selected := make([]material.Lot, 0)
	allocations := make(map[int]int)
	for i := range m.lots[tenant] {
		if m.lots[tenant][i].Kind == kind && m.lots[tenant][i].IsUsable() && remaining > 0 {
			take := m.lots[tenant][i]
			if take.Grams > remaining {
				take.Grams = remaining
			}
			selected = append(selected, take)
			allocations[i] = take.Grams
			remaining -= take.Grams
		}
	}
	if remaining > 0 {
		return nil, apperr.New(apperr.Conflict, "insufficient material")
	}
	for index, reserved := range allocations {
		m.lots[tenant][index].Grams -= reserved
	}
	return selected, nil
}
func (m *MaterialService) AssaySummary(tenant string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	passed := 0
	for _, items := range m.assays {
		for _, a := range items {
			if a.TenantID == tenant {
				n++
				if a.Passes() {
					passed++
				}
			}
		}
	}
	return fmt.Sprintf("%d/%d assays pass", passed, n)
}
func (m *MaterialService) Prune(before time.Time) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	removed := 0
	for tenant, items := range m.lots {
		keep := items[:0]
		for _, v := range items {
			if v.AssayedAt.Before(before) {
				removed++
			} else {
				keep = append(keep, v)
			}
		}
		m.lots[tenant] = keep
	}
	return removed
}
