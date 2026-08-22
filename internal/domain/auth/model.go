package auth

type Role string

const (
	Operator   Role = "operator"
	Supervisor Role = "supervisor"
)

type Principal struct {
	ID, TenantID, Name string
	Role               Role
	SessionID          string
}

func (p Principal) Can(action string) bool {
	if p.Role == Supervisor {
		return true
	}
	switch action {
	case "intake", "inspect", "reserve", "recover":
		return p.Role == Operator
	case "certify", "audit":
		return false
	}
	return false
}
