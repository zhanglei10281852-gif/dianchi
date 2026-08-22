package transport

import (
	"fmt"
	"strings"
)

func ValidateVehicle(v Vehicle) error {
	if v.ID == "" || strings.TrimSpace(v.Plate) == "" {
		return fmt.Errorf("vehicle identity required")
	}
	if v.Capacity <= 0 {
		return fmt.Errorf("vehicle capacity invalid")
	}
	if !v.HazardApproved {
		return fmt.Errorf("vehicle hazard approval required")
	}
	return nil
}
func Transition(from, to Status) error {
	allowed := map[Status][]Status{Planned: {Loading}, Loading: {InTransit}, InTransit: {Arrived}, Arrived: {Closed}, Closed: {}}
	for _, v := range allowed[from] {
		if v == to {
			return nil
		}
	}
	return fmt.Errorf("invalid trip transition")
}
func ValidateStop(s Stop) error {
	if s.ID == "" || s.TripID == "" || s.Location == "" {
		return fmt.Errorf("stop identity incomplete")
	}
	if s.Sequence < 1 {
		return fmt.Errorf("stop sequence invalid")
	}
	return nil
}
