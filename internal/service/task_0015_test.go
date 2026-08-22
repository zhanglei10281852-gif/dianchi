package service

import (
	"context"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/notification"
)

func TestCanceledNotificationCompletionKeepsSendingState(t *testing.T) {
	ctx := context.Background()
	p := auth.Principal{ID: "operator", TenantID: "plant-a", Role: auth.Operator}
	svc := NewNotificationService()
	message := notification.Message{ID: "message-15", Subject: "certificate ready", Body: "lot-15", Channel: notification.Webhook}
	if err := svc.Queue(ctx, p, message); err != nil {
		t.Fatal(err)
	}
	if _, ok := svc.Claim(ctx, time.Now().Add(time.Second)); !ok {
		t.Fatal("queued message was not claimed")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := svc.Complete(canceled, message.ID, nil); err == nil {
		t.Fatal("canceled completion succeeded")
	}
	rows := svc.List(p.TenantID)
	if len(rows) != 1 || rows[0].Status != notification.Sending {
		t.Fatalf("canceled completion changed message: %+v", rows)
	}
	if err := svc.Complete(ctx, message.ID, nil); err != nil {
		t.Fatalf("live completion failed: %v", err)
	}
}
