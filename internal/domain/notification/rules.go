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
func Prepare(candidate Message, now time.Time) (Message, error) {
	if err := Validate(candidate); err != nil {
		return Message{}, err
	}
	candidate.Status = Queued
	candidate.CreatedAt = now
	candidate.NextAttempt = now
	return candidate, nil
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
