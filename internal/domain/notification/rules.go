package notification

import (
	"fmt"
	"strings"
	"time"
)

func Validate(m Message) error {
	if m.ID == "" || m.TenantID == "" {
		return fmt.Errorf("message identity required")
	}
	if strings.TrimSpace(m.Subject) == "" || strings.TrimSpace(m.Body) == "" {
		return fmt.Errorf("message content required")
	}
	switch m.Channel {
	case Email, Webhook, Pager:
	default:
		return fmt.Errorf("unsupported channel")
	}
	return nil
}
func Complete(current Message, deliveryErr error, now time.Time) Message {
	if deliveryErr == nil {
		current.Status = Sent
		return current
	}
	current.Status = Failed
	current.NextAttempt = now.Add(time.Duration(Backoff(current.Attempts)) * time.Second)
	return current
}

func Backoff(attempt int) int64 {
	if attempt < 1 {
		return 1
	}
	delay := int64(1)
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	return delay
}
