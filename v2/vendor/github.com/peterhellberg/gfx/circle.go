package gfx

import (
	"fmt"
	"math"
)

// Circle is a 2D circle. It is defined by two properties:
//  - Center vector
//  - Radius float64
type Circle struct {
	Center Vec
	Radius float64
}

// C returns a new Circle with the given radius and center coordinates.
//
// Note that a negative radius is valid.
func C(center Vec, radius float64) Circle {
	return Circle{
		Center: center,
		Radius: radius,
	}
}

// String returns the string representation of the Circle.
func (c Circle) String() string {
	return fmt.Sprintf("gfx.C(%s, %.2f)", c.Center, c.Radius)
}

// Norm returns the Circle in normalized form - this sets the radius to its absolute value.
func (c Circle) Norm() Circle {
	return Circle{
		Center: c.Center,
		Radius: math.Abs(c.Radius),
	}
}

// Area returns the area of the Circle.
func (c Circle) Area() float64 {
	return math.Pi * math.Pow(c.Radius, 2)
}

// Moved returns the Circle moved by the given vector delta.
func (c Circle) Moved(delta Vec) Circle {
	return Circle{
		Center: c.Center.Add(delta),
		Radius: c.Radius,
	}
}

// Resized returns the Circle resized by the given delta.  The Circles center is use as the anchor.
func (c Circle) Resized(radiusDelta float64) Circle {
	return Circle{
		Center: c.Center,
		Radius: c.Radius + radiusDelta,
	}
}

// Contains checks whether a vector `u` is contained within this Circle (including it's perimeter).
func (c Circle) Contains(u Vec) bool {
	toCenter := c.Center.To(u)
	return c.Radius >= toCenter.Len()
}

// Union returns the minimal Circle which covers both `c` and `d`.
func (c Circle) Union(d Circle) Circle {
	biggerC := maxCircle(c.Norm(), d.Norm())
	smallerC := minCircle(c.Norm(), d.Norm())

	// Get distance between centers
	dist := c.Center.To(d.Center).Len()

	// If the bigger Circle encompasses the smaller one, we have the result
	if dist+smallerC.Radius <= biggerC.Radius {
		return biggerC
	}

	// Calculate radius for encompassing Circle
	r := (dist + biggerC.Radius + smallerC.Radius) / 2

	// Calculate center for encompassing Circle
	theta := .5 + (biggerC.Radius-smallerC.Radius)/(2*dist)
	center := smallerC.Center.Lerp(biggerC.Center, theta)

	return Circle{
		Center: center,
		Radius: r,
	}
}

// Intersect returns the maximal Circle which is covered by both `c` and `d`.
//
// If `c` and `d` don't overlap, this function returns a zero-sized circle at the centerpoint between the two Circle's
// centers.
func (c Circle) Intersect(d Circle) Circle {
	// Check if one of the circles encompasses the other; if so, return that one
	biggerC := maxCircle(c.Norm(), d.Norm())
	smallerC := minCircle(c.Norm(), d.Norm())

	if biggerC.Radius >= biggerC.Center.To(smallerC.Center).Len()+smallerC.Radius {
		return biggerC
	}

	// Calculate the midpoint between the two radii
	// Distance between centers
	dist := c.Center.To(d.Center).Len()
	// Difference between radii
	diff := dist - (c.Radius + d.Radius)
	// Distance from c.Center to the weighted midpoint
	distToMidpoint := c.Radius + 0.5*diff
	// Weighted midpoint
	center := c.Center.Lerp(d.Center, distToMidpoint/dist)

	// No need to calculate radius if the circles do not overlap
	if c.Center.To(d.Center).Len() >= c.Radius+d.Radius {
		return C(center, 0)
	}

	radius := c.Center.To(d.Center).Len() - (c.Radius + d.Radius)

	return Circle{
		Center: center,
		Radius: math.Abs(radius),
	}
}

// IntersectRect returns a minimal required Vector, such that moving the circle by that vector would stop the Circle
// and the Rect intersecting.  This function returns a zero-vector if the Circle and Rect do not overlap, and if only
// the perimeters touch.
//
// This function will return a non-zero vector if:
//  - The Rect contains the Circle, partially or fully
//  - The Circle contains the Rect, partially of fully
func (c Circle) IntersectRect(r Rect) Vec {
	// Checks if the c.Center is not in the diagonal quadrants of the rectangle
	if (r.Min.X <= c.Center.X && c.Center.X <= r.Max.X) || (r.Min.Y <= c.Center.Y && c.Center.Y <= r.Max.Y) {
		// 'grow' the Rect by c.Radius in each orthagonal
		grown := Rect{
			Min: r.Min.Sub(V(c.Radius, c.Radius)),
			Max: r.Max.Add(V(c.Radius, c.Radius)),
		}

		if !grown.Contains(c.Center) {
			// c.Center not close enough to overlap, return zero-vector
			return ZV
		}

		// Get minimum distance to travel out of Rect
		rToC := r.Center().To(c.Center)
		h := c.Radius - math.Abs(rToC.X) + (r.W() / 2)
		v := c.Radius - math.Abs(rToC.Y) + (r.H() / 2)

		if rToC.X < 0 {
			h = -h
		}

		if rToC.Y < 0 {
			v = -v
		}

		// No intersect
		if h == 0 && v == 0 {
			return ZV
		}

		if math.Abs(h) > math.Abs(v) {
			// Vertical distance shorter
			return V(0, v)
		}

		return V(h, 0)
	}

	// The center is in the diagonal quadrants

	// Helper points to make code below easy to read.
	rectTopLeft := V(r.Min.X, r.Max.Y)
	rectBottomRight := V(r.Max.X, r.Min.Y)

	// Check for overlap.
	if !(c.Contains(r.Min) || c.Contains(r.Max) || c.Contains(rectTopLeft) || c.Contains(rectBottomRight)) {
		// No overlap.
		return ZV
	}

	var centerToCorner Vec

	if c.Center.To(r.Min).Len() <= c.Radius {
		// Closest to bottom-left
		centerToCorner = c.Center.To(r.Min)
	}

	if c.Center.To(r.Max).Len() <= c.Radius {
		// Closest to top-right
		centerToCorner = c.Center.To(r.Max)
	}

	if c.Center.To(rectTopLeft).Len() <= c.Radius {
		// Closest to top-left
		centerToCorner = c.Center.To(rectTopLeft)
	}

	if c.Center.To(rectBottomRight).Len() <= c.Radius {
		// Closest to bottom-right
		centerToCorner = c.Center.To(rectBottomRight)
	}

	cornerToCircumferenceLen := c.Radius - centerToCorner.Len()

	return centerToCorner.Unit().Scaled(cornerToCircumferenceLen)
}

// IntersectCircle returns a minimal required Vector, such that moving the circle by that vector would stop the Circle
// and the Rect intersecting.  This function returns a zero-vector if the Circle and Rect do not overlap, and if only
// the perimeters touch.
//
// This function will return a non-zero vector if:
//  - The Rect contains the Circle, partially or fully
//  - The Circle contains the Rect, partially of fully
func (r Rect) IntersectCircle(c Circle) Vec {
	return c.IntersectRect(r).Scaled(-1)
}

// maxCircle will return the larger circle based on the radius.
func maxCircle(c, d Circle) Circle {
	if c.Radius < d.Radius {
		return d
	}
	return c
}

// minCircle will return the smaller circle based on the radius.
func minCircle(c, d Circle) Circle {
	if c.Radius < d.Radius {
		return c
	}
	return d
}
