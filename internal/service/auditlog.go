package service

import (
	"context"
	"encoding/json"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/audit"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"sync"
	"time"
)

type AuditLog struct {
	mu     sync.RWMutex
	events []audit.Event
}

func NewAuditLog() *AuditLog { return &AuditLog{events: make([]audit.Event, 0)} }
func (a *AuditLog) Write(ctx context.Context, p auth.Principal, object, action, result, request string, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, audit.Event{ID: token(), TenantID: p.TenantID, ActorID: p.ID, ObjectType: "domain", ObjectID: object, Action: action, Result: result, RequestID: request, Payload: string(data), CreatedAt: time.Now().UTC()})
	return nil
}
func (a *AuditLog) Query(ctx context.Context, p auth.Principal, object string) []audit.Event {
	if ctx.Err() != nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]audit.Event, 0)
	for _, e := range a.events {
		if e.TenantID == p.TenantID && (object == "" || e.ObjectID == object) {
			out = append(out, e)
		}
	}
	return out
}
func (a *AuditLog) Last(ctx context.Context, p auth.Principal, object string) (audit.Event, bool) {
	rows := a.Query(ctx, p, object)
	if len(rows) == 0 {
		return audit.Event{}, false
	}
	return rows[len(rows)-1], true
}
