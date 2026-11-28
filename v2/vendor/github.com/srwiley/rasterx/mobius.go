// Copyright 2022 by the rasterx Authors. All rights reserved.
// Created by S.R.Wiley 2022
//
// This file implements Mobius transformations, i.e.
// M(z) = (az+b)/(cz+d)
// where a, b, c, d, and z are complex numbers.
// The real part of z maps to the x path axis and
// the imaginary part of z maps to the y axis.

package rasterx

import (
	"golang.org/x/image/math/fixed"
)

// Mobius represents a mapping of the complex plane
// to itself of the form  M(z) = (az+b)/(cz+d)
// Non degenerate forms of this mapping are one to one
// and differentiable.
type Mobius struct {
	A, B, C, D complex128
}

// Transform returns the Mobius m transform applied to z
func (m Mobius) Transform(z complex128) complex128 {
	return (m.A*z + m.B) / (m.C*z + m.D)
}

// TFixed transforms a fixed.Point26_6 by the mobius
func (m Mobius) TFixed(x fixed.Point26_6) fixed.Point26_6 {
	ct := m.Transform(complex(float64(x.X)/64, float64(x.Y)/64))
	return fixed.Point26_6{X: fixed.Int26_6(real(ct) * 64), Y: fixed.Int26_6(imag(ct) * 64)}
}

// MatrixAdder is an adder that applies mobius M to all points
type MobiusAdder struct {
	Adder
	M Mobius
}

// Start starts a new path
func (t *MobiusAdder) Start(a fixed.Point26_6) {
	t.Adder.Start(t.M.TFixed(a))
}

// Line adds a linear segment to the current curve.
func (t *MobiusAdder) Line(b fixed.Point26_6) {
	t.Adder.Line(t.M.TFixed(b))
}

// QuadBezier adds a quadratic segment to the current curve.
func (t *MobiusAdder) QuadBezier(b, c fixed.Point26_6) {
	t.Adder.QuadBezier(t.M.TFixed(b), t.M.TFixed(c))
}

// CubeBezier adds a cubic segment to the current curve.
func (t *MobiusAdder) CubeBezier(b, c, d fixed.Point26_6) {
	t.Adder.CubeBezier(t.M.TFixed(b), t.M.TFixed(c), t.M.TFixed(d))
}
