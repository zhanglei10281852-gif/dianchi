package operator

import (
	"fmt"
	"strings"
	"time"
)

func Validate(o Operator) error {
	if o.ID == "" || o.TenantID == "" || strings.TrimSpace(o.Name) == "" {
		return fmt.Errorf("operator identity incomplete")
	}
	if !o.Active {
		return fmt.Errorf("operator inactive")
	}
	if o.Shift == "" {
		return fmt.Errorf("shift required")
	}
	return nil
}
func GrantCertification(current Operator, cert Certification, expires, now time.Time) (Operator, error) {
	if !current.Active {
		return Operator{}, fmt.Errorf("operator inactive")
	}
	if cert == "" || !now.Before(expires) {
		return Operator{}, fmt.Errorf("certification expiry invalid")
	}
	current.Certifications = MergeCertifications(nil, current.Certifications)
	current.Certifications[cert] = expires
	return current, nil
}

func MergeCertifications(base, extra map[Certification]time.Time) map[Certification]time.Time {
	out := map[Certification]time.Time{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
