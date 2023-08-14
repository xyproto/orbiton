package main

import "math"

// abs returns the absolute value of the given int
func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// distance returns the distance between two points, given int
func distance(x1, y1, x2, y2 int) float64 {
	x1f := float64(x1)
	y1f := float64(y1)
	x2f := float64(x2)
	y2f := float64(y2)
	return math.Sqrt((x1f*x1f - x2f*x2f) + (y1f*y1f - y2f*y2f))
}
