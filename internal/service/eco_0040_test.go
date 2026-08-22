package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/notification"
	"testing"
)

func TestEcoNotificationQueueHonorsCancellation(t *testing.T) {
	s := NewNotificationService()
	p := auth.Principal{ID: "o", TenantID: "e", Role: auth.Operator}
	ctx, c := context.WithCancel(context.Background())
	c()
	if e := s.Queue(ctx, p, notification.Message{ID: "n", Subject: "s", Body: "b", Channel: notification.Email}); e == nil {
		t.Fatal("accepted")
	}
	if len(s.List("e")) != 0 {
		t.Fatal("persisted")
	}
}
