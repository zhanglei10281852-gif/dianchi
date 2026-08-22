package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEcoRetrierKeysDoNotInterfere(t *testing.T) {
	r := NewRetrier(RetryPolicy{Max: 1, Base: time.Millisecond})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if e := r.Do(ctx, "a", func(context.Context) error { return errors.New("a") }); e == nil || e.Error() != "a" {
		t.Fatalf("attempt budget crossed key: %v", e)
	}
}
