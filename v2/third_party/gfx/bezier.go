package gfx

import "math"

// CubicBezierCurve returns a slice of vectors representing a cubic bezier curve
func CubicBezierCurve(p0, p1, p2, p3 Vec, inc float64) []Vec {
	if inc <= 0 {
		return nil
	}

	var curve []Vec

	for u := 0.0; u <= 1.0; u += inc {
		n := 1 - u
		a := math.Pow(n, 3)
		b := math.Pow(n, 2)
		c := math.Pow(u, 2)
		d := math.Pow(u, 3)
		ub := 3 * u * b
		cn := 3 * c * n

		curve = append(curve, V(
			a*p0.X+ub*p1.X+cn*p2.X+d*p3.X,
			a*p0.Y+ub*p1.Y+cn*p2.Y+d*p3.Y,
		))
	}

	return curve
}
