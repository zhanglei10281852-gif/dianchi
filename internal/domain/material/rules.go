package material

import "fmt"

func ValidateWeight(kind Kind, grams int) error {
	if kind == "" {
		return fmt.Errorf("material kind required")
	}
	if grams <= 0 {
		return fmt.Errorf("material weight must be positive")
	}
	if grams > 1000000 {
		return fmt.Errorf("material weight exceeds lot capacity")
	}
	return nil
}
func ValidateAssay(a Assay) error {
	if err := ValidateWeight(a.Kind, 1); err != nil {
		return err
	}
	if a.PurityPPM < 0 || a.PurityPPM > 1000000 {
		return fmt.Errorf("purity out of range")
	}
	if a.MoisturePPM < 0 {
		return fmt.Errorf("moisture out of range")
	}
	return nil
}
func PrepareAssay(candidate Assay) (Assay, error) {
	if err := ValidateAssay(candidate); err != nil {
		return Assay{}, err
	}
	candidate.Grade = Classify(candidate.PurityPPM, candidate.MoisturePPM)
	return candidate, nil
}

func Classify(purity, moisture int) Grade {
	if purity < 950000 || moisture > 500 {
		return GradeReject
	}
	if purity >= 990000 && moisture <= 100 {
		return GradeA
	}
	if purity >= 970000 {
		return GradeB
	}
	return GradeC
}
func AcceptableFor(kind Kind, grade Grade) bool {
	switch kind {
	case Lithium, Nickel, Cobalt:
		return grade == GradeA || grade == GradeB
	case Copper, Graphite:
		return grade != GradeReject
	default:
		return false
	}
}
func NormalizeKind(v string) Kind {
	switch v {
	case "Li", "lithium":
		return Lithium
	case "Ni", "nickel":
		return Nickel
	case "Co", "cobalt":
		return Cobalt
	case "Cu", "copper":
		return Copper
	case "graphite":
		return Graphite
	default:
		return Kind(v)
	}
}
