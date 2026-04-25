//go:build !tinygo
// +build !tinygo

package gfx

// IntAbs returns the absolute value of x.
func IntAbs(x int) int {
	if x > 0 {
		return x
	}

	return -x
}

// IntMin returns the smaller of x or y.
func IntMin(x, y int) int {
	if x < y {
		return x
	}

	return y
}

// IntMax returns the larger of x or y.
func IntMax(x, y int) int {
	if x > y {
		return x
	}

	return y
}

// IntClamp returns x clamped to the interval [min, max].
//
// If x is less than min, min is returned.
// If x is more than max, max is returned. Otherwise, x is returned.
func IntClamp(x, min, max int) int {
	switch {
	case x < min:
		return min
	case x > max:
		return max
	default:
		return x
	}
}
