package service

import (
	"context"
	"errors"
	"github.com/zhanglei10281852-gif/dianchi/internal/repository"
	"time"
)

type Publisher interface {
	Publish(context.Context, string, string) error
}
type OutboxWorker struct {
	Store       repository.Store
	Publisher   Publisher
	Clock       func() time.Time
	MaxAttempts int
}

func (w OutboxWorker) RunOnce(ctx context.Context) error {
	id, agg, payload, err := w.Store.ClaimOutbox(ctx, w.Clock())
	if err != nil {
		return err
	}
	err = w.Publisher.Publish(ctx, agg, payload)
	return w.Store.MarkOutbox(ctx, id, nil)
}
func (w OutboxWorker) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				continue
			}
		}
	}
}
