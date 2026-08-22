package service

import (
	"context"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"github.com/zhanglei10281852-gif/dianchi/internal/pagination"
	"sort"
	"strings"
	"time"
)

type Report struct {
	Tenant      string
	GeneratedAt time.Time
	ByState     map[battery.State]int
	ByChemistry map[string]int
	Expiring    []battery.Lot
}

func (s *Service) Report(ctx context.Context, p auth.Principal, within time.Duration) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	rows, err := s.List(ctx, p, pagination.Query{Limit: 100})
	if err != nil {
		return Report{}, err
	}
	out := Report{Tenant: p.TenantID, GeneratedAt: s.Clock.Now(), ByState: map[battery.State]int{}, ByChemistry: map[string]int{}, Expiring: make([]battery.Lot, 0)}
	cut := s.Clock.Now().Add(within)
	for _, l := range rows.Items {
		out.ByState[l.State]++
		out.ByChemistry[l.Chemistry]++
		if l.ExpiresAt != nil && l.ExpiresAt.Before(cut) {
			out.Expiring = append(out.Expiring, l)
		}
	}
	sort.Slice(out.Expiring, func(i, j int) bool { return out.Expiring[i].ReceivedAt.Before(out.Expiring[j].ReceivedAt) })
	return out, nil
}
func FormatReport(r Report) string {
	parts := make([]string, 0, len(r.ByState))
	for state, n := range r.ByState {
		parts = append(parts, string(state)+"="+itoa(n))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	buf := ""
	for v > 0 {
		buf = string(rune('0'+v%10)) + buf
		v /= 10
	}
	return sign + buf
}
