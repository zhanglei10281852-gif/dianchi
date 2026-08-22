package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/notification"
)

func TestQueueDuplicateIDPreservesExistingMessage(t *testing.T) {
	svc := NewNotificationService()
	ctx := context.Background()
	principal := auth.Principal{ID: "op-1", TenantID: "tenant-a", Name: "op", Role: auth.Operator}

	original := notification.Message{
		ID:      "msg-pickup-001",
		Subject: "提货提醒：批次 LOT-001",
		Body:    "请于今日 18:00 前至 7 号站点提货",
		Channel: notification.Email,
	}
	if err := svc.Queue(ctx, principal, original); err != nil {
		t.Fatalf("queue original: %v", err)
	}

	// Producer retries the same message ID but carries different content.
	// The service must reject the duplicate without overwriting the original.
	duplicate := original
	duplicate.Subject = "提货提醒：批次 LOT-999（已被覆盖）"
	duplicate.Body = "请前往 3 号站点提货"
	err := svc.Queue(ctx, principal, duplicate)

	var e *apperr.Error
	if !errors.As(err, &e) || e.Code != apperr.Conflict {
		t.Fatalf("expected conflict error, got %v", err)
	}

	stored := svc.List(principal.TenantID)
	if len(stored) != 1 {
		t.Fatalf("stored count=%d", len(stored))
	}
	if stored[0].Subject != original.Subject || stored[0].Body != original.Body {
		t.Fatalf("original message overwritten: subject=%q body=%q", stored[0].Subject, stored[0].Body)
	}
}

func TestQueueDistinctIDsCoexist(t *testing.T) {
	svc := NewNotificationService()
	ctx := context.Background()
	principal := auth.Principal{ID: "op-1", TenantID: "tenant-a", Name: "op", Role: auth.Operator}

	base := notification.Message{Channel: notification.Email, Subject: "s", Body: "b"}
	for i, id := range []string{"msg-1", "msg-2", "msg-3"} {
		m := base
		m.ID = id
		if err := svc.Queue(ctx, principal, m); err != nil {
			t.Fatalf("queue %s: %v", id, err)
		}
		_ = i
	}
	if got := len(svc.List(principal.TenantID)); got != 3 {
		t.Fatalf("coexisting count=%d", got)
	}
}

func TestQueueCancelledContext(t *testing.T) {
	svc := NewNotificationService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	principal := auth.Principal{ID: "op-1", TenantID: "tenant-a", Name: "op", Role: auth.Operator}
	err := svc.Queue(ctx, principal, notification.Message{ID: "msg-x", Channel: notification.Email, Subject: "s", Body: "b"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestQueueRejectsInvalidMessage(t *testing.T) {
	svc := NewNotificationService()
	ctx := context.Background()
	principal := auth.Principal{ID: "op-1", TenantID: "tenant-a", Name: "op", Role: auth.Operator}
	err := svc.Queue(ctx, principal, notification.Message{ID: "msg-y", Channel: notification.Email, Subject: "", Body: "b"})
	var e *apperr.Error
	if !errors.As(err, &e) || e.Code != apperr.Invalid {
		t.Fatalf("expected invalid error, got %v", err)
	}
}

func TestClaimAndCompleteLifecycle(t *testing.T) {
	svc := NewNotificationService()
	ctx := context.Background()
	principal := auth.Principal{ID: "op-1", TenantID: "tenant-a", Name: "op", Role: auth.Operator}
	if err := svc.Queue(ctx, principal, notification.Message{ID: "msg-lc", Channel: notification.Email, Subject: "s", Body: "b"}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	m, ok := svc.Claim(ctx, now)
	if !ok {
		t.Fatal("expected claim")
	}
	if m.Status != notification.Sending || m.Attempts != 1 {
		t.Fatalf("claim state=%s attempts=%d", m.Status, m.Attempts)
	}
	if err := svc.Complete(ctx, m.ID, nil); err != nil {
		t.Fatal(err)
	}
	stored := svc.List(principal.TenantID)[0]
	if stored.Status != notification.Sent {
		t.Fatalf("status=%s", stored.Status)
	}
}
