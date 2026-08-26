package battery

import "fmt"

type Measurement struct {
	Voltage, Temperature, Resistance float64
	At                               string
}

func (m Measurement) Valid() bool {
	return m.Voltage > 0 && m.Temperature >= -50 && m.Temperature <= 100 && m.Resistance >= 0
}
func (m Measurement) Risk() string {
	if m.Temperature > 60 || m.Resistance > 100 {
		return "high"
	}
	if m.Temperature > 45 || m.Resistance > 50 {
		return "medium"
	}
	return "low"
}
func ValidateMeasurement(m Measurement) error {
	if !m.Valid() {
		return fmt.Errorf("measurement outside physical limits")
	}
	return nil
}
