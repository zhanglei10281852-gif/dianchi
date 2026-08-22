package audit

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/audit"
	"sort"
	"time"
)

func Filter(events []audit.Event, action string, from, to time.Time) []audit.Event {
	out := make([]audit.Event, 0)
	for _, e := range events {
		if action != "" && e.Action != action {
			continue
		}
		if !from.IsZero() && e.CreatedAt.Before(from) {
			continue
		}
		if !to.IsZero() && e.CreatedAt.After(to) {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}
func Purge(ctx context.Context, events []audit.Event, before time.Time) []audit.Event {
	if ctx.Err() != nil {
		return events
	}
	out := events[:0]
	for _, e := range events {
		if !e.CreatedAt.Before(before) {
			out = append(out, e)
		}
	}
	return out
}
