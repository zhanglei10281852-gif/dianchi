package compliance

import "time"

type Status string

const (
	Draft     Status = "draft"
	Submitted Status = "submitted"
	Approved  Status = "approved"
	Rejected  Status = "rejected"
	Expired   Status = "expired"
)

type Certificate struct {
	ID, TenantID, LotID, Serial, Issuer string
	Status                              Status
	IssuedAt                            time.Time
	ExpiresAt                           time.Time
	Evidence                            []string
}
type Requirement struct {
	Code, Description string
	Required          bool
}
type Review struct {
	ID, CertificateID, ReviewerID, Decision, Notes string
	ReviewedAt                                     time.Time
}

func (c Certificate) Active(now time.Time) bool {
	return c.Status == Approved && now.Before(c.ExpiresAt)
}
func (c Certificate) CanSubmit() bool  { return c.Status == Draft && len(c.Evidence) > 0 }
func (c Certificate) CanApprove() bool { return c.Status == Submitted && c.Issuer != "" }
