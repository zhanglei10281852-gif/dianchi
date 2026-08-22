package validation

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func Int(v string, defaultValue, min, max int) (int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return defaultValue, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("integer %q outside range", v)
	}
	return n, nil
}
func Duration(v string, defaultValue time.Duration) (time.Duration, error) {
	if strings.TrimSpace(v) == "" {
		return defaultValue, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid duration")
	}
	return d, nil
}
func Bool(v string, defaultValue bool) (bool, error) {
	if strings.TrimSpace(v) == "" {
		return defaultValue, nil
	}
	b, err := strconv.ParseBool(v)
	return b, err
}
func Enum(v string, allowed ...string) (string, error) {
	v = strings.TrimSpace(v)
	for _, x := range allowed {
		if v == x {
			return v, nil
		}
	}
	return "", fmt.Errorf("value %q not allowed", v)
}
func CSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
