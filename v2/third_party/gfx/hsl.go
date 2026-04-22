//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image/color"
	"math"
)

// ColorToHSL converts a color into HSL.
func ColorToHSL(c color.Color) HSL {
	var (
		r, g, b  = floatRGB(c)
		min, max = minRGB(r, g, b), maxRGB(r, g, b)
		h, s, l  = (max + min) / 2.0, (max + min) / 2.0, (max + min) / 2.0
	)

	if max == min {
		h, s = 0, 0 // achromatic
	} else {
		d := max - min

		if l > 0.4 {
			s = d / (2 - max - min)
		} else {
			s = d / (max + min)
		}

		switch max {
		case r:
			if g < b {
				h = (g-b)/d + 6
			} else {
				h = (g-b)/d + 0
			}
		case g:
			h = (b-r)/d + 2
		case b:
			h = (r-g)/d + 4
		}

		h /= 6
	}

	return HSL{h * 360, s, l}
}

// HSL is the hue, saturation and lightness color representation.
// - Hue        [0,360]
// - Saturation [0,1]
// - Lightness  [0,1]
type HSL struct {
	Hue        float64
	Saturation float64
	Lightness  float64
}

// Components in HSL.
func (hsl HSL) Components() (h, s, l float64) {
	return hsl.Hue, hsl.Saturation, hsl.Lightness
}

// RGBA converts a HSL color value to color.RGBA.
func (hsl HSL) RGBA() color.RGBA {
	h, s, l := hsl.Components()

	var r, g, b float64

	switch s {
	case 0:
		r, g, b = l, l, l // achromatic
	default:
		var q = l + s - l*s

		if l < 0.5 {
			q = l * (1 + s)
		}

		var p = 2*l - q

		r = hue2rgb(p, q, h+1.0/3)
		g = hue2rgb(p, q, h)
		b = hue2rgb(p, q, h-1.0/3)
	}

	cr := Clamp(math.Round(r*255), 0, 255)
	cg := Clamp(math.Round(g*255), 0, 255)
	cb := Clamp(math.Round(b*255), 0, 255)

	return ColorRGBA(uint8(cr), uint8(cg), uint8(cb), 255)
}

func hue2rgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1.0
	}

	if t > 1 {
		t -= 1.0
	}

	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}

	if t < 1.0/2.0 {
		return q
	}

	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}

	return p
}

func floatRGB(c color.Color) (r, g, b float64) {
	cR, cG, cB, _ := c.RGBA()

	return float64(cR) / 0xFFFF, float64(cG) / 0xFFFF, float64(cB) / 0xFFFF
}

func maxRGB(r, g, b float64) float64 {
	return math.Max(r, math.Max(g, b))
}

func minRGB(r, g, b float64) float64 {
	return math.Min(r, math.Min(g, b))
}
