package main

import "math"

// abs returns the absolute value of the given int
func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// umin finds the smallest uint
func umin(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// distance returns the distance between two points, given int
func distance(x1, y1, x2, y2 int) float64 {
	return math.Sqrt(float64((x1-x2)*(x1-x2) + (y1-y2)*(y1-y2)))
}
