//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image/color"
	"math"
)

// ColorToXYZ converts a color into XYZ.
//
// R, G and B (Standard RGB) input range = 0 ÷ 255
// X, Y and Z output refer to a D65/2° standard illuminant.
func ColorToXYZ(c color.Color) XYZ {
	r, g, b := floatRGB(c)

	if r > 0.04045 {
		r = math.Pow((r+0.055)/1.055, 2.4)
	} else {
		r = r / 12.92
	}

	if g > 0.04045 {
		g = math.Pow((g+0.055)/1.055, 2.4)
	} else {
		g = g / 12.92
	}

	if b > 0.04045 {
		b = math.Pow((b+0.055)/1.055, 2.4)
	} else {
		b = b / 12.92
	}

	r = r * 100.0
	g = g * 100.0
	b = b * 100.0

	return XYZ{
		X: (r * 0.4124) + (g * 0.3576) + (b * 0.1805),
		Y: (r * 0.2126) + (g * 0.7152) + (b * 0.0722),
		Z: (r * 0.0193) + (g * 0.1192) + (b * 0.9505),
	}
}

// XYZ color space.
type XYZ struct {
	X float64
	Y float64
	Z float64
}

// XYZReference values of a perfect reflecting diffuser.
type XYZReference struct {
	A   XYZ // Incandescent/tungsten
	B   XYZ // Old direct sunlight at noon
	C   XYZ // Old daylight
	D50 XYZ // ICC profile PCS
	D55 XYZ // Mid-morning daylight
	D65 XYZ // Daylight, sRGB, Adobe-RGB
	D75 XYZ // North sky daylight
	E   XYZ // Equal energy
	F1  XYZ // Daylight Fluorescent
	F2  XYZ // Cool fluorescent
	F3  XYZ // White Fluorescent
	F4  XYZ // Warm White Fluorescent
	F5  XYZ // Daylight Fluorescent
	F6  XYZ // Lite White Fluorescent
	F7  XYZ // Daylight fluorescent, D65 simulator
	F8  XYZ // Sylvania F40, D50 simulator
	F9  XYZ // Cool White Fluorescent
	F10 XYZ // Ultralume 50, Philips TL85
	F11 XYZ // Ultralume 40, Philips TL84
	F12 XYZ // Ultralume 30, Philips TL83
}

var (
	// XYZReference2 for CIE 1931 2° Standard Observer
	XYZReference2 = XYZReference{
		A:   XYZ{109.850, 100.000, 35.585},
		B:   XYZ{99.0927, 100.000, 85.313},
		C:   XYZ{98.074, 100.000, 118.232},
		D50: XYZ{96.422, 100.000, 82.521},
		D55: XYZ{95.682, 100.000, 92.149},
		D65: XYZ{95.047, 100.000, 108.883},
		D75: XYZ{94.972, 100.000, 122.638},
		E:   XYZ{100.000, 100.000, 100.000},
		F1:  XYZ{92.834, 100.000, 103.665},
		F2:  XYZ{99.187, 100.000, 67.395},
		F3:  XYZ{103.754, 100.000, 49.861},
		F4:  XYZ{109.147, 100.000, 38.813},
		F5:  XYZ{90.872, 100.000, 98.723},
		F6:  XYZ{97.309, 100.000, 60.191},
		F7:  XYZ{95.044, 100.000, 108.755},
		F8:  XYZ{96.413, 100.000, 82.333},
		F9:  XYZ{100.365, 100.000, 67.868},
		F10: XYZ{96.174, 100.000, 81.712},
		F11: XYZ{100.966, 100.000, 64.370},
		F12: XYZ{108.046, 100.000, 39.228},
	}

	// XYZReference10 for CIE 1964 10° Standard Observer
	XYZReference10 = XYZReference{
		A:   XYZ{111.144, 100.000, 35.200},
		B:   XYZ{99.178, 100.000, 84.3493},
		C:   XYZ{97.285, 100.000, 116.145},
		D50: XYZ{96.720, 100.000, 81.427},
		D55: XYZ{95.799, 100.000, 90.926},
		D65: XYZ{94.811, 100.000, 107.304},
		D75: XYZ{94.416, 100.000, 120.641},
		E:   XYZ{100.000, 100.000, 100.000},
		F1:  XYZ{94.791, 100.000, 103.191},
		F2:  XYZ{103.280, 100.000, 69.026},
		F3:  XYZ{108.968, 100.000, 51.965},
		F4:  XYZ{114.961, 100.000, 40.963},
		F5:  XYZ{93.369, 100.000, 98.636},
		F6:  XYZ{102.148, 100.000, 62.074},
		F7:  XYZ{95.792, 100.000, 107.687},
		F8:  XYZ{97.115, 100.000, 81.135},
		F9:  XYZ{102.116, 100.000, 67.826},
		F10: XYZ{99.001, 100.000, 83.134},
		F11: XYZ{103.866, 100.000, 65.627},
		F12: XYZ{111.428, 100.000, 40.353},
	}
)
