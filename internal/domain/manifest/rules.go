package manifest

import (
	"fmt"
	"strings"
)

func Validate(m Manifest) error {
	if m.ID == "" || m.TenantID == "" {
		return fmt.Errorf("manifest identity required")
	}
	if strings.TrimSpace(m.Origin) == "" || strings.TrimSpace(m.Destination) == "" {
		return fmt.Errorf("manifest endpoints required")
	}
	if m.Origin == m.Destination {
		return fmt.Errorf("manifest endpoints must differ")
	}
	for _, i := range m.Items {
		if i.Weight <= 0 {
			return fmt.Errorf("manifest item weight invalid")
		}
	}
	return nil
}
func Transition(from, to Status) error {
	allowed := map[Status][]Status{Prepared: {Sealed, Cancelled}, Sealed: {Dispatched, Cancelled}, Dispatched: {Received}, Received: {}, Cancelled: {}}
	for _, v := range allowed[from] {
		if v == to {
			return nil
		}
	}
	return fmt.Errorf("invalid manifest transition")
}

func ValidateMove(current Manifest, target Status) error {
	if current.ID == "" || current.TenantID == "" {
		return fmt.Errorf("manifest identity required")
	}
	return Transition(current.Status, target)
}
func NormalizeClass(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v == "" {
		return "UNCLASSIFIED"
	}
	return v
}
