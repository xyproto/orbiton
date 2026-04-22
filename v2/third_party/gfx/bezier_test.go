package gfx

import "testing"

func BenchmarkCubicBezierCurve(b *testing.B) {
	var (
		p0  = V(16, 192)
		p1  = V(32, 8)
		p2  = V(192, 244)
		p3  = V(240, 128)
		inc = 0.0009
	)

	for i := 0; i < b.N; i++ {
		CubicBezierCurve(p0, p1, p2, p3, inc)
	}
}
