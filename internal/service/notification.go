package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/notification"
	"sync"
	"time"
)

type NotificationService struct {
	mu    sync.Mutex
	items map[string]notification.Message
}

func NewNotificationService() *NotificationService {
	return &NotificationService{items: map[string]notification.Message{}}
}
func (n *NotificationService) Queue(ctx context.Context, p auth.Principal, m notification.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := notification.Validate(m); err != nil {
		return apperr.Wrap(apperr.Invalid, "notification", err)
	}
	m.TenantID = p.TenantID
	m.Status = notification.Queued
	m.CreatedAt = time.Now().UTC()
	m.NextAttempt = m.CreatedAt
	n.mu.Lock()
	defer n.mu.Unlock()
	n.items[m.ID] = m
	return nil
}
func (n *NotificationService) Claim(ctx context.Context, now time.Time) (notification.Message, bool) {
	if err := ctx.Err(); err != nil {
		return notification.Message{}, false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	for id, m := range n.items {
		if m.Ready(now) {
			m.Status = notification.Sending
			m.Attempts++
			n.items[id] = m
			return m, true
		}
	}
	return notification.Message{}, false
}
func (n *NotificationService) Complete(ctx context.Context, id string, err error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	m, ok := n.items[id]
	if !ok {
		return apperr.New(apperr.NotFound, "notification missing")
	}
	if err == nil {
		m.Status = notification.Sent
	} else {
		m.Status = notification.Failed
		m.NextAttempt = time.Now().UTC().Add(time.Duration(notification.Backoff(m.Attempts)) * time.Second)
	}
	n.items[id] = m
	return nil
}
func (n *NotificationService) List(tenant string) []notification.Message {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]notification.Message, 0)
	for _, m := range n.items {
		if m.TenantID == tenant {
			out = append(out, m)
		}
	}
	return out
}
