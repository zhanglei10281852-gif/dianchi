package incident

import (
	"fmt"
	"strings"
)

func Transition(from, to Status) error {
	allowed := map[Status][]Status{Open: {Investigating, Contained}, Investigating: {Contained, Resolved}, Contained: {Resolved}, Resolved: {}}
	for _, v := range allowed[from] {
		if v == to {
			return nil
		}
	}
	return fmt.Errorf("invalid incident transition %s -> %s", from, to)
}
func ValidateAction(current Incident, description string) error {
	if current.Status == Resolved {
		return fmt.Errorf("resolved incident cannot accept actions")
	}
	if len(strings.TrimSpace(description)) < 3 {
		return fmt.Errorf("action description is required")
	}
	return nil
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
