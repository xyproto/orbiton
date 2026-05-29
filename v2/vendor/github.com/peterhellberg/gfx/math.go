package gfx

import (
	"cmp"
	"math"
)

// Numeric is a type constraint that permits any built-in numeric type
// (integer or floating point). It is exported so callers can reuse it
// when building on top of the generic helpers below.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Signed is a type constraint that permits any signed numeric type.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

// Mathematical constants.
const (
	Pi = 3.14159265358979323846264338327950288419716939937510582097494459 // https://oeis.org/A000796
)

// MathMin returns the smaller of x or y.
func MathMin(x, y float64) float64 {
	return math.Min(x, y)
}

// MathMax returns the larger of x or y.
func MathMax(x, y float64) float64 {
	return math.Max(x, y)
}

// MathAbs returns the absolute value of x.
func MathAbs(x float64) float64 {
	return math.Abs(x)
}

// MathSqrt returns the square root of x.
func MathSqrt(x float64) float64 {
	return math.Sqrt(x)
}

// MathSin returns the sine of the radian argument x.
func MathSin(x float64) float64 {
	return math.Sin(x)
}

// MathSinh returns the hyperbolic sine of x.
func MathSinh(x float64) float64 {
	return math.Sinh(x)
}

// MathCos returns the cosine of the radian argument x.
func MathCos(x float64) float64 {
	return math.Cos(x)
}

// MathCosh returns the hyperbolic cosine of x.
func MathCosh(x float64) float64 {
	return math.Cosh(x)
}

// MathAtan returns the arctangent, in radians, of x.
func MathAtan(x float64) float64 {
	return math.Atan(x)
}

// MathTan returns the tangent of the radian argument x.
func MathTan(x float64) float64 {
	return math.Tan(x)
}

// MathCeil returns the least integer value greater than or equal to x.
func MathCeil(x float64) float64 {
	return math.Ceil(x)
}

// MathFloor returns the greatest integer value less than or equal to x.
func MathFloor(x float64) float64 {
	return math.Floor(x)
}

// MathHypot returns Sqrt(p*p + q*q), taking care to avoid unnecessary overflow and underflow.
func MathHypot(p, q float64) float64 {
	return math.Hypot(p, q)
}

// MathPow returns x**y, the base-x exponential of y.
func MathPow(x, y float64) float64 {
	return math.Pow(x, y)
}

// MathLog returns the natural logarithm of x.
func MathLog(x float64) float64 {
	return math.Log(x)
}

// MathRound returns the nearest integer, rounding half away from zero.
func MathRound(x float64) float64 {
	return math.Round(x)
}

// Sign returns -1 for values < 0, 0 for 0, and 1 for values > 0.
func Sign[T Signed](x T) T {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	default:
		return 0
	}
}

// Clamp returns x clamped to the interval [min, max].
//
// If x is less than min, min is returned. If x is more than max, max is returned. Otherwise, x is
// returned.
func Clamp[T cmp.Ordered](x, min, max T) T {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// Lerp does linear interpolation between two values. The interpolation
// is computed in float64; when T is an integer type the result is
// truncated toward zero on the way back.
func Lerp[T Numeric](a, b T, t float64) T {
	return T(float64(a) + (float64(b)-float64(a))*t)
}
