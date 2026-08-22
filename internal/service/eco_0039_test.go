package service

import (
	"context"
	"errors"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/notification"
	"testing"
	"time"
)

func TestEcoNotificationFailureRemainsRetryable(t *testing.T) {
	s := NewNotificationService()
	p := auth.Principal{ID: "o", TenantID: "e", Role: auth.Operator}
	if e := s.Queue(context.Background(), p, notification.Message{ID: "n", Subject: "s", Body: "b", Channel: notification.Email}); e != nil {
		t.Fatal(e)
	}
	m, ok := s.Claim(context.Background(), time.Now())
	if !ok {
		t.Fatal("claim")
	}
	if e := s.Complete(context.Background(), m.ID, errors.New("reject")); e != nil {
		t.Fatal(e)
	}
	rows := s.List("e")
	if len(rows) != 1 || rows[0].Status != notification.Failed {
		t.Fatalf("rows=%+v", rows)
	}
}
