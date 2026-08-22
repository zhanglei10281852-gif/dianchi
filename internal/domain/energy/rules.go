package energy

import (
	"fmt"
	"math"
)

func Validate(r Reading) error {
	if r.ID == "" || r.TenantID == "" || r.LotID == "" {
		return fmt.Errorf("reading identity required")
	}
	if math.IsNaN(r.Voltage) || math.IsNaN(r.Current) {
		return fmt.Errorf("reading is NaN")
	}
	if !r.Safe() {
		return fmt.Errorf("reading outside safety range")
	}
	return nil
}
func Aggregate(readings []Reading) (float64, error) {
	if len(readings) == 0 {
		return 0, fmt.Errorf("no readings")
	}
	sum := 0.0
	for _, r := range readings {
		if err := Validate(r); err != nil {
			return 0, err
		}
		sum += r.Power()
	}
	return sum / float64(len(readings)), nil
}
