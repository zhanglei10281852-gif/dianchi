package validation

import (
	"fmt"
	"regexp"
	"strings"
)

var codePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{3,31}$`)

func LotCode(code string) error {
	if !codePattern.MatchString(code) {
		return fmt.Errorf("lot code format invalid")
	}
	return nil
}
func Tenant(tenant string) error {
	if strings.TrimSpace(tenant) == "" {
		return fmt.Errorf("tenant required")
	}
	if len(tenant) > 64 {
		return fmt.Errorf("tenant too long")
	}
	return nil
}
func Station(station string) error {
	if strings.TrimSpace(station) == "" {
		return fmt.Errorf("station required")
	}
	return nil
}
func PositiveFields(values ...int) error {
	for _, v := range values {
		if v < 0 {
			return fmt.Errorf("negative value")
		}
	}
	return nil
}
func All(errors ...error) error {
	msgs := make([]string, 0)
	for _, err := range errors {
		if err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	return fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
}
func NormalizeNote(note string) string {
	return strings.TrimSpace(strings.ReplaceAll(note, "\x00", ""))
}
func RequireDistinct(values ...string) error {
	seen := map[string]bool{}
	for _, v := range values {
		if seen[v] {
			return fmt.Errorf("duplicate value %q", v)
		}
		seen[v] = true
	}
	return nil
}
