package service

import (
	"context"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/notification"
)

func TestDuplicateNotificationPreservesOriginalMessage(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	svc := NewNotificationService()
	original := notification.Message{ID: "message-14", Subject: "pickup ready", Body: "lot-14", Channel: notification.Email}
	if err := svc.Queue(ctx, p, original); err != nil {
		t.Fatal(err)
	}
	duplicate := original
	duplicate.Subject = "tampered subject"
	if err := svc.Queue(ctx, p, duplicate); err == nil {
		t.Fatal("duplicate notification was accepted")
	}
	rows := svc.List(p.TenantID)
	if len(rows) != 1 || rows[0].Subject != original.Subject {
		t.Fatalf("duplicate replaced original message: %+v", rows)
	}
	second := original
	second.ID = "message-14-b"
	if err := svc.Queue(ctx, p, second); err != nil {
		t.Fatalf("independent notification failed: %v", err)
	}
}
