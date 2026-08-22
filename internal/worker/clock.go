package worker

import (
	"context"
	"strings"
	"time"
)

func ParseJob(value string) (string, string, bool) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
func JobContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}

func RunWithDeadline(ctx context.Context, d time.Duration, fn func(context.Context) error) error {
	child, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	return fn(child)
}
func Backoff(attempt int, base time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	out := base
	for i := 0; i < attempt; i++ {
		out *= 2
	}
	return out
}
