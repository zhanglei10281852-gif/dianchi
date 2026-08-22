package safety

import (
	"fmt"
	"strings"
	"time"
)

func Validate(f Finding) error {
	if f.ID == "" || f.LotID == "" || f.TenantID == "" {
		return fmt.Errorf("finding identity required")
	}
	if strings.TrimSpace(f.Code) == "" {
		return fmt.Errorf("finding code required")
	}
	if f.Limit <= 0 {
		return fmt.Errorf("finding limit invalid")
	}
	return nil
}
func Assess(measured, limit float64) Level {
	if measured > limit*1.2 {
		return Danger
	}
	if measured > limit {
		return Watch
	}
	return Clear
}
func Resolve(f Finding) Finding {
	if f.ResolvedAt == nil {
		now := time.Now().UTC()
		f.ResolvedAt = &now
	}
	if f.Level == Danger {
		f.Level = Watch
	}
	return f
}
