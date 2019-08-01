package dotstar

import (
	"testing"
)

func TestColourSetup1(t *testing.T) {
	newClr := NewColourFromStr("#01020308")
	if newClr.R != 1 || newClr.G != 2 || newClr.B != 3 || newClr.L != 8 {
		t.Errorf("Got colour %v expected #01020308\n", newClr)
	}
}

func TestColourSetup2(t *testing.T) {
	newClr := NewColourFromStr("#FF000080")
	if newClr.R != 255 || newClr.G != 0 || newClr.B != 0 || newClr.L != 128 {
		t.Errorf("Got colour %v expected #FF000080\n", newClr)
	}
}
