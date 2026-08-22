package battery

import "fmt"

type ChemistryProfile struct {
	Name                                         string
	NominalVoltage                               float64
	HazardClass                                  string
	RequiresDryRoom                              bool
	AllowedTemperatureMin, AllowedTemperatureMax float64
}

var Profiles = map[string]ChemistryProfile{"LFP": {Name: "LFP", NominalVoltage: 3.2, HazardClass: "9", RequiresDryRoom: true, AllowedTemperatureMin: -10, AllowedTemperatureMax: 45}, "NMC": {Name: "NMC", NominalVoltage: 3.7, HazardClass: "9", RequiresDryRoom: true, AllowedTemperatureMin: -5, AllowedTemperatureMax: 40}, "LMO": {Name: "LMO", NominalVoltage: 3.7, HazardClass: "9", RequiresDryRoom: false, AllowedTemperatureMin: -10, AllowedTemperatureMax: 50}}

func Profile(name string) (ChemistryProfile, error) {
	p, ok := Profiles[name]
	if !ok {
		return ChemistryProfile{}, fmt.Errorf("profile not found")
	}
	return p, nil
}
func WithinTemperature(name string, temp float64) bool {
	p, err := Profile(name)
	return err == nil && temp >= p.AllowedTemperatureMin && temp <= p.AllowedTemperatureMax
}
