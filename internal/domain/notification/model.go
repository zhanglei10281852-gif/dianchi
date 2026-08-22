package notification

import "time"

type Channel string

const (
	Email   Channel = "email"
	Webhook Channel = "webhook"
	Pager   Channel = "pager"
)

type Status string

const (
	Queued  Status = "queued"
	Sending Status = "sending"
	Sent    Status = "sent"
	Failed  Status = "failed"
)

type Message struct {
	ID, TenantID, Subject, Body string
	Channel                     Channel
	Status                      Status
	Attempts                    int
	NextAttempt                 time.Time
	CreatedAt                   time.Time
}

func (m Message) Retryable(max int) bool { return m.Status == Failed && m.Attempts < max }
func (m Message) Ready(now time.Time) bool {
	return m.Status == Queued || m.Status == Failed && !m.NextAttempt.After(now)
}
