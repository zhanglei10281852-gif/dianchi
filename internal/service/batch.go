package service

import (
	"context"
	"fmt"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"sync"
	"time"
)

type BatchItem struct {
	Code, Chemistry string
	Hazard          int
	ExpiresAt       *time.Time
}
type BatchResult struct {
	Created               []battery.Lot
	Failures              map[string]error
	StartedAt, FinishedAt time.Time
}
type BatchService struct {
	Lifecycle *Service
	Workers   int
}

func (b BatchService) Import(ctx context.Context, p auth.Principal, items []BatchItem, request string) (BatchResult, error) {
	if !p.Can("intake") {
		return BatchResult{}, apperr.New(apperr.Forbidden, "role cannot import")
	}
	if len(items) == 0 {
		return BatchResult{}, apperr.Invalidf("batch is empty")
	}
	started := time.Now().UTC()
	out := BatchResult{Created: make([]battery.Lot, 0, len(items)), Failures: map[string]error{}, StartedAt: started}
	workers := b.Workers
	if workers < 1 {
		workers = 1
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	for _, item := range items {
		item := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				out.Failures[item.Code] = ctx.Err()
				return
			}
			defer func() { <-sem }()
			lot, err := b.Lifecycle.Intake(ctx, p, item.Code, item.Chemistry, item.Hazard, item.ExpiresAt, request+"/"+item.Code)
			if err != nil {
				out.Failures[item.Code] = err
			} else {
				out.Created = append(out.Created, lot)
			}
		}()
	}
	wg.Wait()
	out.FinishedAt = time.Now().UTC()
	return out, nil
}
func (b BatchService) RetryFailed(ctx context.Context, p auth.Principal, failures map[string]error) int {
	if err := ctx.Err(); err != nil {
		return 0
	}
	n := 0
	for code := range failures {
		if _, err := b.Lifecycle.Intake(ctx, p, code, "LFP", 1, nil, "retry/"+code); err == nil {
			n++
		}
	}
	return n
}
func (b BatchService) Summary(r BatchResult) string {
	return fmt.Sprintf("created=%d failed=%d duration=%s", len(r.Created), len(r.Failures), r.FinishedAt.Sub(r.StartedAt))
}
