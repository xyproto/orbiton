package gfx

import (
	"math"
	"testing"
)

func inDelta(t *testing.T, expected, actual, delta float64) bool {
	t.Helper()

	if math.IsNaN(expected) && math.IsNaN(actual) {
		return true
	}

	if math.IsNaN(expected) {
		t.Error("Expected must not be NaN")
		return false
	}

	if math.IsNaN(actual) {
		t.Errorf("Expected %v with delta %v, but was NaN", expected, delta)
		return false
	}

	if dt := expected - actual; dt < -delta || dt > delta {
		t.Errorf("Max difference between %v and %v allowed is %v, but difference was %v", expected, actual, delta, dt)
		return false
	}

	return true
}
