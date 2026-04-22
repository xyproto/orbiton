package gfx

import "math"

const degToRad = math.Pi / 180

// Degrees of arc.
type Degrees float64

// Radians convert degrees to radians.
func (d Degrees) Radians() float64 {
	return float64(d) * degToRad
}
