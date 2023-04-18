package gfx

import (
	"fmt"
	"math"
)

// Vec3 is a 3D vector type with X, Y and Z coordinates.
//
// Create vectors with the V3 constructor:
//
//   u := gfx.V3(1, 2, 3)
//   v := gfx.V3(8, -3, 4)
//
type Vec3 struct {
	X, Y, Z float64
}

// ZV3 is the zero Vec3
var ZV3 = Vec3{0, 0, 0}

// V3 is shorthand for Vec3{X: x, Y: y, Z: z}.
func V3(x, y, z float64) Vec3 {
	return Vec3{x, y, z}
}

// IV3 returns a new 3D vector based on the given int x, y, z values.
func IV3(x, y, z int) Vec3 {
	return Vec3{float64(x), float64(y), float64(z)}
}

// String returns the string representation of the vector u.
func (u Vec3) String() string {
	return fmt.Sprintf("gfx.V3(%v, %v, %v)", u.X, u.Y, u.Z)
}

// XYZ returns the components of the vector in three return values.
func (u Vec3) XYZ() (x, y, z float64) {
	return u.X, u.Y, u.Z
}

// Eq checks the equality of two vectors.
func (u Vec3) Eq(v Vec3) bool {
	return u.X == v.X && u.Y == v.Y && u.Z == v.Z
}

// Vec returns a Vec with X, Y coordinates.
func (u Vec3) Vec() Vec {
	return V(u.X, u.Y)
}

// Add returns the sum of vectors u and v.
func (u Vec3) Add(v Vec3) Vec3 {
	return Vec3{
		u.X + v.X,
		u.Y + v.Y,
		u.Z + v.Z,
	}
}

// AddXYZ returns the sum of x, y and z added to u.
func (u Vec3) AddXYZ(x, y, z float64) Vec3 {
	return Vec3{
		u.X + x,
		u.Y + y,
		u.Z + z,
	}
}

// Sub returns the difference betweeen vectors u and v.
func (u Vec3) Sub(v Vec3) Vec3 {
	return Vec3{
		u.X - v.X,
		u.Y - v.Y,
		u.Z - v.Z,
	}
}

// Scaled returns the vector u multiplied by c.
func (u Vec3) Scaled(s float64) Vec3 {
	return Vec3{
		u.X * s,
		u.Y * s,
		u.Z * s,
	}
}

// ScaledXYZ returns the component-wise multiplication of two vectors.
func (u Vec3) ScaledXYZ(v Vec3) Vec3 {
	return Vec3{
		u.X * v.X,
		u.Y * v.Y,
		u.Z * v.Z,
	}
}

// Len returns the length (euclidian norm) of a vector.
func (u Vec3) Len() float64 {
	return math.Sqrt(u.SqLen())
}

// Div returns the vector v/s.
func (u Vec3) Div(s float64) Vec3 {
	return Vec3{
		u.X / s,
		u.Y / s,
		u.Z / s,
	}
}

// Dot returns the dot product of vectors u and v.
func (u Vec3) Dot(v Vec3) float64 {
	return u.X*v.X + u.Y*v.Y + u.Z*v.Z
}

// SqDist returns the square of the euclidian distance between two vectors.
func (u Vec3) SqDist(v Vec3) float64 {
	return u.Sub(v).SqLen()
}

// Dist returns the euclidian distance between two vectors.
func (u Vec3) Dist(v Vec3) float64 {
	return u.Sub(v).Len()
}

// SqLen returns the square of the length (euclidian norm) of a vector.
func (u Vec3) SqLen() float64 {
	return u.Dot(u)
}

// Unit returns the normalized vector of a vector.
func (u Vec3) Unit() Vec3 {
	return u.Div(u.Len())
}

// Map applies the function f to the x, y and z components of the vector u
// and returns the modified vector.
func (u Vec3) Map(f func(float64) float64) Vec3 {
	return Vec3{
		f(u.X),
		f(u.Y),
		f(u.Z),
	}
}

// Lerp returns the linear interpolation between v and w by amount t.
// The amount t is usually a value between 0 and 1. If t=0 v will be
// returned; if t=1 w will be returned.
func (u Vec3) Lerp(v Vec3, t float64) Vec3 {
	return Vec3{
		Lerp(u.X, v.X, t),
		Lerp(u.Y, v.Y, t),
		Lerp(u.Z, v.Z, t),
	}
}
