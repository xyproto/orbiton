package gfx

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
)

// ZR is the zero image.Rectangle.
var ZR = image.ZR

// Rect is a 2D rectangle aligned with the axes of the coordinate system. It is defined by two
// points, Min and Max.
//
// The invariant should hold, that Max's components are greater or equal than Min's components
// respectively.
type Rect struct {
	Min, Max Vec
}

// NewRect creates a new Rect.
func NewRect(min, max Vec) Rect {
	return Rect{
		Min: min,
		Max: max,
	}
}

// R returns a new Rect given the Min and Max coordinates.
//
// Note that the returned rectangle is not automatically normalized.
func R(minX, minY, maxX, maxY float64) Rect {
	return NewRect(Vec{minX, minY}, Vec{maxX, maxY})
}

// BoundsToRect converts an image.Rectangle to a Rect.
func BoundsToRect(ir image.Rectangle) Rect {
	return R(float64(ir.Min.X), float64(ir.Min.Y), float64(ir.Max.X), float64(ir.Max.Y))
}

// BoundsCenter returns the vector in the center of an image.Rectangle
func BoundsCenter(ir image.Rectangle) Vec {
	return BoundsToRect(ir).Center()
}

// BoundsCenterOrigin returns the center origin for the given image.Rectangle and z value.
func BoundsCenterOrigin(ir image.Rectangle, v Vec, z float64) Vec3 {
	return BoundsToRect(ir).CenterOrigin(v, z)
}

// CenterOrigin returns a Vec3 based on Rect.Center()
// scaled by v, and its Z component set to the provided z.
func (r Rect) CenterOrigin(v Vec, z float64) Vec3 {
	return r.Center().ScaledXY(v).Vec3(z)
}

// String returns the string representation of the Rect.
//
//   r := gfx.R(100, 50, 200, 300)
//   r.String()     // returns "gfx.R(100, 50, 200, 300)"
//   fmt.Println(r) // gfx.R(100, 50, 200, 300)
func (r Rect) String() string {
	return fmt.Sprintf("gfx.R(%v, %v, %v, %v)", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
}

// Norm returns the Rect in normal form, such that Max is component-wise greater or equal than Min.
func (r Rect) Norm() Rect {
	return Rect{
		Min: Vec{
			math.Min(r.Min.X, r.Max.X),
			math.Min(r.Min.Y, r.Max.Y),
		},
		Max: Vec{
			math.Max(r.Min.X, r.Max.X),
			math.Max(r.Min.Y, r.Max.Y),
		},
	}
}

// W returns the width of the Rect.
func (r Rect) W() float64 {
	return r.Max.X - r.Min.X
}

// H returns the height of the Rect.
func (r Rect) H() float64 {
	return r.Max.Y - r.Min.Y
}

// Size returns the vector of width and height of the Rect.
func (r Rect) Size() Vec {
	return V(r.W(), r.H())
}

// Area returns the area of r. If r is not normalized, area may be negative.
func (r Rect) Area() float64 {
	return r.W() * r.H()
}

// Center returns the position of the center of the Rect.
func (r Rect) Center() Vec {
	return r.Min.Lerp(r.Max, 0.5)
}

// Moved returns the Rect moved (both Min and Max) by the given vector delta.
func (r Rect) Moved(delta Vec) Rect {
	return Rect{
		Min: r.Min.Add(delta),
		Max: r.Max.Add(delta),
	}
}

// Resized returns the Rect resized to the given size while keeping the position of the given
// anchor.
//
//   r.Resized(r.Min, size)      // resizes while keeping the position of the lower-left corner
//   r.Resized(r.Max, size)      // same with the top-right corner
//   r.Resized(r.Center(), size) // resizes around the center
//
// This function does not make sense for resizing a rectangle of zero area and will panic. Use
// ResizedMin in the case of zero area.
func (r Rect) Resized(anchor, size Vec) Rect {
	if r.W()*r.H() == 0 {
		panic(fmt.Errorf("(%T).Resize: zero area", r))
	}
	fraction := Vec{size.X / r.W(), size.Y / r.H()}
	return Rect{
		Min: anchor.Add(r.Min.Sub(anchor).ScaledXY(fraction)),
		Max: anchor.Add(r.Max.Sub(anchor).ScaledXY(fraction)),
	}
}

// ResizedMin returns the Rect resized to the given size while keeping the position of the Rect's
// Min.
//
// Sizes of zero area are safe here.
func (r Rect) ResizedMin(size Vec) Rect {
	return Rect{
		Min: r.Min,
		Max: r.Min.Add(size),
	}
}

// Contains checks whether a vector u is contained within this Rect (including it's borders).
func (r Rect) Contains(u Vec) bool {
	return r.Min.X <= u.X && u.X <= r.Max.X && r.Min.Y <= u.Y && u.Y <= r.Max.Y
}

// Overlaps checks whether one Rect overlaps another Rect.
func (r Rect) Overlaps(s Rect) bool {
	return r.Intersect(s) != Rect{}
}

// Union returns the minimal Rect which covers both r and s. Rects r and s must be normalized.
func (r Rect) Union(s Rect) Rect {
	return R(
		math.Min(r.Min.X, s.Min.X),
		math.Min(r.Min.Y, s.Min.Y),
		math.Max(r.Max.X, s.Max.X),
		math.Max(r.Max.Y, s.Max.Y),
	)
}

// Intersect returns the maximal Rect which is covered by both r and s. Rects r and s must be normalized.
//
// If r and s don't overlap, this function returns R(0, 0, 0, 0).
func (r Rect) Intersect(s Rect) Rect {
	t := R(
		math.Max(r.Min.X, s.Min.X),
		math.Max(r.Min.Y, s.Min.Y),
		math.Min(r.Max.X, s.Max.X),
		math.Min(r.Max.Y, s.Max.Y),
	)
	if t.Min.X >= t.Max.X || t.Min.Y >= t.Max.Y {
		return Rect{}
	}
	return t
}

// Bounds returns the bounds of the rectangle.
func (r Rect) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: r.Min.Pt(),
		Max: r.Max.Pt(),
	}
}

// Draw draws Rect to src over dst, at the zero point.
func (r Rect) Draw(dst draw.Image, src image.Image) {
	draw.Draw(dst, r.Bounds(), src, ZP, draw.Src)
}

// DrawColor draws Rect with a uniform color on dst.
func (r Rect) DrawColor(dst draw.Image, c color.Color) {
	draw.Draw(dst, r.Bounds(), NewUniform(c), ZP, draw.Src)
}

// DrawColorOver draws Rect with a uniform color over dst.
func (r Rect) DrawColorOver(dst draw.Image, c color.Color) {
	draw.Draw(dst, r.Bounds(), NewUniform(c), ZP, draw.Over)
}

// EachVec calls the provided function for each vec in the given direction.
func (r Rect) EachVec(dir Vec, fn func(p Vec)) {
	if dir.X == 0 {
		dir.X = 1
	}

	if dir.Y == 0 {
		dir.Y = 1
	}

	switch {
	case dir.X > 0 && dir.Y < 0:
		for y := r.Max.Y - 1; y >= r.Min.Y; y += dir.Y {
			for x := r.Min.X; x < r.Max.X; x += dir.X {
				fn(V(x, y))
			}
		}
	case dir.X < 0 && dir.Y < 0:
		for y := r.Max.Y - 1; y >= r.Min.Y; y += dir.Y {
			for x := r.Max.X - 1; x >= r.Min.X; x += dir.X {
				fn(V(x, y))
			}
		}
	case dir.X < 0 && dir.Y > 0:
		for y := r.Min.Y; y < r.Max.Y; y += dir.Y {
			for x := r.Max.X - 1; x >= r.Min.X; x += dir.X {
				fn(V(x, y))
			}
		}
	default:
		for y := r.Min.Y; y < r.Max.Y; y += dir.Y {
			for x := r.Min.X; x < r.Max.X; x += dir.X {
				fn(V(x, y))
			}
		}
	}
}
