//go:build !tinygo
// +build !tinygo

package gfx

import "sort"

// NewLinearScaler creates a new linear scaler.
func NewLinearScaler() LinearScaler {
	return LinearScaler{
		d: Domain{0, 1},
		r: Range{0, 1},
	}
}

// LinearScaler can scale domain values to a range values.
type LinearScaler struct {
	d Domain
	r Range
}

// Domain returns a LinearScaler with the given domain.
func (ls LinearScaler) Domain(d ...float64) LinearScaler {
	if len(d) > 0 {
		ls.d = d
		sort.Float64s(ls.d)
	}

	return ls
}

// Range returns a LinearScaler with the given range.
func (ls LinearScaler) Range(r ...float64) LinearScaler {
	if len(r) > 0 {
		ls.r = r
	}

	return ls
}

// ScaleFloat64 from domain to range.
//
// OLD PERCENT = (x - OLD MIN) / (OLD MAX - OLD MIN)
// NEW X = ((NEW MAX - NEW MIN) * OLD PERCENT) + NEW MIN
func (ls LinearScaler) ScaleFloat64(x float64) float64 {
	op := (x - ls.d.Min()) / (ls.d.Max() - ls.d.Min())
	nx := ((ls.r.Last() - ls.r.First()) * op) + ls.r.First()

	return nx
}

// Domain of values.
type Domain []float64

// Min value in the Domain.
func (d Domain) Min() float64 {
	return d[0]
}

// Max value in the Domain.
func (d Domain) Max() float64 {
	return d[len(d)-1]
}

// Range of values.
type Range []float64

// First value in the Range.
func (r Range) First() float64 {
	return r[0]
}

// Last value in the Range.
func (r Range) Last() float64 {
	return r[len(r)-1]
}

func interpolateFloat64s(a, b float64) func(float64) float64 {
	return func(t float64) float64 {
		return a*(1-t) + b*t
	}
}
