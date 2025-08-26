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
	return math.Sqrt(float64((x1-x2)*(x1-x2) + (y1-y2)*(y1-y2)))
}
