package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/notification"
	"sort"
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
	m.TenantID = p.TenantID
	candidate, err := notification.Prepare(m, time.Now().UTC())
	if err != nil {
		return apperr.Wrap(apperr.Invalid, "notification", err)
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.items[m.ID]; exists {
		return apperr.New(apperr.Conflict, "notification already queued")
	}
	n.items[m.ID] = candidate
	return nil
}
func (n *NotificationService) Claim(ctx context.Context, now time.Time) (notification.Message, bool) {
	if err := ctx.Err(); err != nil {
		return notification.Message{}, false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	ids := make([]string, 0, len(n.items))
	for id := range n.items {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		left, right := n.items[ids[i]], n.items[ids[j]]
		if left.NextAttempt.Equal(right.NextAttempt) {
			return left.ID < right.ID
		}
		return left.NextAttempt.Before(right.NextAttempt)
	})
	for _, id := range ids {
		m := n.items[id]
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
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}
