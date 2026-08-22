package compliance

import "fmt"

func Validate(c Certificate) error {
	if c.ID == "" || c.TenantID == "" || c.LotID == "" {
		return fmt.Errorf("certificate identity incomplete")
	}
	if c.Serial == "" {
		return fmt.Errorf("certificate serial required")
	}
	if c.ExpiresAt.Before(c.IssuedAt) {
		return fmt.Errorf("certificate expires before issue")
	}
	return nil
}
func Transition(from, to Status) error {
	allowed := map[Status][]Status{Draft: {Submitted}, Submitted: {Approved, Rejected}, Rejected: {Draft}, Approved: {Expired}}
	for _, v := range allowed[from] {
		if v == to {
			return nil
		}
	}
	return fmt.Errorf("invalid compliance transition %s -> %s", from, to)
}
func RequiredSatisfied(req []Requirement, provided map[string]string) bool {
	for _, r := range req {
		if r.Required && provided[r.Code] == "" {
			return false
		}
	}
	return true
}
func MergeEvidence(base, extra []string) []string {
	out := make([]string, 0, len(base)+len(extra))
	seen := map[string]bool{}
	for _, v := range append(base, extra...) {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
