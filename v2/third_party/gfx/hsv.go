//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image/color"
	"math"
)

// SortByHue sorts based on (HSV) Hue.
func (p Palette) SortByHue() {
	p.Sort(func(i, j int) bool {
		return ColorToHSV(p[i]).Hue > ColorToHSV(p[i]).Hue
	})
}

// ColorToHSV converts a color into HSV.
func ColorToHSV(c color.Color) HSV {
	var (
		r, g, b    = floatRGB(c)
		min, max   = minRGB(r, g, b), maxRGB(r, g, b)
		h, s, v, d = max, max, max, max - min
	)

	if max == 0 {
		s = 0
	} else {
		s = d / max
	}

	if max == min {
		h = 0 // achromatic
	} else {
		switch max {
		case r:
			if g < b {
				h = (g-b)/d + 6.0
			} else {
				h = (g - b) / d
			}
		case g:
			h = (b-r)/d + 2.0
		case b:
			h = (r-g)/d + 4.0
		}
	}

	return HSV{h, s * 100.0, v * 100.0}
}

// HSV is the hue, saturation and value color representation.
// - Hue        [0,360]
// - Saturation [0,1]
// - Value      [0,1]
type HSV struct {
	Hue        float64
	Saturation float64
	Value      float64
}

// Components in HSV.
func (hsv HSV) Components() (h, s, v float64) {
	return hsv.Hue, hsv.Saturation, hsv.Value
}

// RGBA converts a HSV color value to color.RGBA.
func (hsv HSV) RGBA() color.RGBA {
	h, s, v := hsv.Components()

	hprime := h / 60.0

	var r, g, b float64

	c := v * s
	x := c * math.Abs(math.Remainder(hprime, 2))
	m := v - c

	switch {
	case hprime >= 0 && hprime < 1:
		r = c
		g = x
		b = 0
	case hprime >= 1 && hprime < 2:
		r = x
		g = c
		b = 0
	case hprime >= 2 && hprime < 3:
		r = 0
		g = c
		b = x
	case hprime >= 3 && hprime < 4:
		r = 0
		g = x
		b = c
	case hprime >= 4 && hprime < 5:
		r = x
		g = 0
		b = c
	case hprime >= 5 && hprime < 6:
		r = c
		g = 0
		b = x
	}

	return ColorRGBA(
		uint8(math.Round((r+m)*255)),
		uint8(math.Round((g+m)*255)),
		uint8(math.Round((b+m)*255)),
		255,
	)
}
