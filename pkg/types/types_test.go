package types

import "testing"

func TestPackageExists(t *testing.T) {
	// Trivial test to verify the project compiles and tests run
	if 1+1 != 2 {
		t.Fatal("math is broken")
	}
}
