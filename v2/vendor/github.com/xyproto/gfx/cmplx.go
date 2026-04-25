package gfx

import (
	"image/color"
	"math/cmplx"
)

// CmplxSin returns the sine of x.
func CmplxSin(x complex128) complex128 {
	return cmplx.Sin(x)
}

// CmplxSinh returns the hyperbolic sine of x.
func CmplxSinh(x complex128) complex128 {
	return cmplx.Sinh(x)
}

// CmplxCos returns the cosine of x.
func CmplxCos(x complex128) complex128 {
	return cmplx.Cos(x)
}

// CmplxCosh returns the hyperbolic cosine of x.
func CmplxCosh(x complex128) complex128 {
	return cmplx.Cosh(x)
}

// CmplxTan returns the tangent of x.
func CmplxTan(x complex128) complex128 {
	return cmplx.Tan(x)
}

// CmplxTanh returns the hyperbolic tangent of x.
func CmplxTanh(x complex128) complex128 {
	return cmplx.Tanh(x)
}

// CmplxPow returns x**y, the base-x exponential of y.
func CmplxPow(x, y complex128) complex128 {
	return cmplx.Pow(x, y)
}

// CmplxSqrt returns the square root of x.
// The result r is chosen so that real(r) ≥ 0 and imag(r) has the same sign as imag(x).
func CmplxSqrt(x complex128) complex128 {
	return cmplx.Sqrt(x)
}

// CmplxPhase returns the phase (also called the argument) of x.
// The returned value is in the range [-Pi, Pi].
func CmplxPhase(x complex128) float64 {
	return cmplx.Phase(x)
}

// CmplxPhaseAt returns the color at the phase of the given complex128 value.
func (p Palette) CmplxPhaseAt(z complex128) color.Color {
	t := CmplxPhase(z)/Pi + 1

	if t > 1 {
		t = 2 - t
	}

	return p.At(t)
}
