package incident

import "fmt"

func Transition(from, to Status) error {
	allowed := map[Status][]Status{Open: {Investigating, Contained}, Investigating: {Contained, Resolved}, Contained: {Resolved}, Resolved: {}}
	for _, v := range allowed[from] {
		if v == to {
			return nil
		}
	}
	return fmt.Errorf("invalid incident transition %s -> %s", from, to)
}
func ValidateMove(current Incident, target Status) error {
	if current.ID == "" || current.TenantID == "" {
		return fmt.Errorf("incident identity incomplete")
	}
	return Transition(current.Status, target)
}

func Validate(i Incident) error {
	if i.ID == "" || i.TenantID == "" || i.Summary == "" {
		return fmt.Errorf("incident fields required")
	}
	if i.Severity == "" {
		return fmt.Errorf("incident severity required")
	}
	return nil
}
