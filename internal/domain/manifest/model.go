package manifest

import "time"

type Status string

const (
	Prepared   Status = "prepared"
	Sealed     Status = "sealed"
	Dispatched Status = "dispatched"
	Received   Status = "received"
	Cancelled  Status = "cancelled"
)

type Item struct {
	ID, ManifestID, LotID, TenantID string
	Weight                          int
	HazardClass                     string
}
type Manifest struct {
	ID, TenantID, Origin, Destination string
	Status                            Status
	CreatedBy                         string
	CreatedAt                         time.Time
	SealedAt, ReceivedAt              *time.Time
	Items                             []Item
}

func (m Manifest) CanSeal() bool     { return m.Status == Prepared && len(m.Items) > 0 }
func (m Manifest) CanDispatch() bool { return m.Status == Sealed }
func (m Manifest) CanReceive() bool  { return m.Status == Dispatched }
func (m Manifest) TotalWeight() int {
	n := 0
	for _, i := range m.Items {
		n += i.Weight
	}
	return n
}
