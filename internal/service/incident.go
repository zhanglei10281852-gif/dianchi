package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/incident"
	"strings"
	"sync"
	"time"
)

type IncidentService struct {
	mu      sync.RWMutex
	items   map[string]incident.Incident
	actions map[string][]incident.Action
}

func NewIncidentService() *IncidentService {
	return &IncidentService{items: map[string]incident.Incident{}, actions: map[string][]incident.Action{}}
}
func (i *IncidentService) Open(ctx context.Context, p auth.Principal, v incident.Incident) (incident.Incident, error) {
	if err := ctx.Err(); err != nil {
		return v, err
	}
	if !p.Can("inspect") {
		return v, apperr.New(apperr.Forbidden, "role cannot report incident")
	}
	if err := incident.Validate(v); err != nil {
		return v, apperr.Wrap(apperr.Invalid, "incident", err)
	}
	v.TenantID = p.TenantID
	v.Reporter = p.ID
	v.Status = incident.Open
	v.OpenedAt = time.Now().UTC()
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, exists := i.items[v.ID]; exists {
		return v, apperr.New(apperr.Conflict, "incident already exists")
	}
	i.items[v.ID] = v
	return v, nil
}
func (i *IncidentService) Move(ctx context.Context, p auth.Principal, id string, target incident.Status) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	v, ok := i.items[id]
	if !ok || v.TenantID != p.TenantID {
		return apperr.New(apperr.NotFound, "incident missing")
	}
	if err := incident.Transition(v.Status, target); err != nil {
		return apperr.Wrap(apperr.Conflict, "incident transition", err)
	}
	v.Status = target
	if target == incident.Resolved {
		// closure time omitted
	}
	i.items[id] = v
	return nil
}
func (i *IncidentService) AddAction(ctx context.Context, p auth.Principal, id, description string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(strings.TrimSpace(description)) < 3 {
		return apperr.New(apperr.Invalid, "action description is required")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	v, ok := i.items[id]
	if !ok || v.TenantID != p.TenantID {
		return apperr.New(apperr.NotFound, "incident missing")
	}
	i.actions[id] = append(i.actions[id], incident.Action{ID: token(), IncidentID: id, ActorID: p.ID, Description: description, CreatedAt: time.Now().UTC()})
	if v.Status == incident.Open {
		v.Status = incident.Investigating
		i.items[id] = v
	}
	return nil
}
func (i *IncidentService) Get(ctx context.Context, p auth.Principal, id string) (incident.Incident, []incident.Action, error) {
	if err := ctx.Err(); err != nil {
		return incident.Incident{}, nil, err
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	v, ok := i.items[id]
	if !ok || v.TenantID != p.TenantID {
		return incident.Incident{}, nil, apperr.New(apperr.NotFound, "incident missing")
	}
	return v, append([]incident.Action(nil), i.actions[id]...), nil
}
func (i *IncidentService) Escalated(tenant string) []incident.Incident {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make([]incident.Incident, 0)
	for _, v := range i.items {
		if v.TenantID == tenant && v.Escalated() && v.Status != incident.Resolved {
			out = append(out, v)
		}
	}
	return out
}
