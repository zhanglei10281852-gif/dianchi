package battery

import "testing"

func TestChemistryNormalization(t *testing.T) {
	cases := []struct{ in, want string }{{"lfp", "LFP"}, {" Nmc ", "NMC"}, {"LMO", "LMO"}}
	for _, tc := range cases {
		got, err := NormalizeChemistry(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("%q -> %q %v", tc.in, got, err)
		}
	}
	if _, err := NormalizeChemistry("unknown"); err == nil {
		t.Fatal("unknown chemistry accepted")
	}
}
func TestAllowedTransitions(t *testing.T) {
	if err := NextState(Received, Inspected); err != nil {
		t.Fatal(err)
	}
	if err := NextState(Inspected, Reserved); err != nil {
		t.Fatal(err)
	}
	if err := NextState(Reserved, Dismantling); err != nil {
		t.Fatal(err)
	}
	if err := NextState(Dismantling, Recovered); err != nil {
		t.Fatal(err)
	}
	if err := NextState(Recovered, Certified); err != nil {
		t.Fatal(err)
	}
	if err := NextState(Received, Certified); err == nil {
		t.Fatal("illegal transition accepted")
	}
}
func TestLotPredicates(t *testing.T) {
	l := Lot{State: Received}
	if !l.CanInspect() {
		t.Fatal()
	}
	l.State = Inspected
	if !l.CanReserve() {
		t.Fatal()
	}
	l.HazardScore = 70
	if l.CanReserve() {
		t.Fatal()
	}
	l.State = Dismantling
	if !l.CanRecover() {
		t.Fatal()
	}
	l.State = Recovered
	if !l.CanCertify() {
		t.Fatal()
	}
}
