package palgen

import (
	"image/color"
)

// A basic 16-color palette
var BasicPalette16 = [16][3]byte{
	{0x0, 0x0, 0x0},    // 0
	{191, 0x0, 0x0},    // 1
	{0x0, 191, 0x0},    // 2
	{191, 191, 0x0},    // 3
	{0x0, 0x0, 191},    // 4
	{191, 0x0, 191},    // 5
	{0x0, 191, 191},    // 6
	{191, 191, 191},    // 7
	{0x40, 0x40, 0x40}, // 8
	{0xff, 0x40, 0x40}, // 9
	{0x40, 0xff, 0x40}, // 10
	{0xff, 0xff, 0x40}, // 11
	{96, 96, 0xff},     // 12
	{0xff, 0x40, 0xff}, // 13
	{0x40, 0xff, 0xff}, // 14
	{0xff, 0xff, 0xff}, // 15
}

// BasicPalette can return a basic 16 color palette
func BasicPalette() (pal color.Palette) {
	for _, rgb := range BasicPalette16 {
		pal = append(pal, color.NRGBA{rgb[0], rgb[1], rgb[2], 255})
	}
	return pal
}
