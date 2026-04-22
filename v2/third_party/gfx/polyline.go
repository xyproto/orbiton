package gfx

import "math"

// Polyline is a slice of polygons forming a line.
type Polyline []Polygon

// NewPolyline constructs a slice of line polygons.
func NewPolyline(p Polygon, t float64) Polyline {
	l := len(p)

	if l < 2 {
		return []Polygon{}
	}

	var pl Polyline

	for i := range p[:l-1] {
		pl = append(pl, newLinePolygon(p[i], p[i+1], t))
	}

	return pl
}

func polylineFromTo(from, to Vec, t float64) Polygon {
	return NewPolyline(Polygon{from, to}, t)[0]
}

func newLinePolygon(from, to Vec, t float64) Polygon {
	a := from.To(to).Angle()

	return Polygon{
		V(from.X+t*math.Cos(a+math.Pi/2), from.Y+t*math.Sin(a+math.Pi/2)),
		V(from.X+t*math.Cos(a-math.Pi/2), from.Y+t*math.Sin(a-math.Pi/2)),

		V(to.X+t*math.Cos(a-math.Pi/2), to.Y+t*math.Sin(a-math.Pi/2)),
		V(to.X+t*math.Cos(a+math.Pi/2), to.Y+t*math.Sin(a+math.Pi/2)),
	}
}
