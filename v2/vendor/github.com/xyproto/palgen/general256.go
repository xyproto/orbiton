package palgen

import (
	"image/color"
)

// GeneralPalette256 is a an OK standard palette
var GeneralPalette256 = [256][3]byte{
	{0, 0, 0},
	{0, 0, 102},
	{0, 0, 204},
	{0, 23, 51},
	{0, 23, 153},
	{0, 23, 255},
	{0, 46, 0},
	{0, 46, 102},
	{0, 46, 204},
	{0, 69, 51},
	{0, 69, 153},
	{0, 69, 255},
	{0, 92, 0},
	{0, 92, 102},
	{0, 92, 204},
	{0, 115, 51},
	{0, 115, 153},
	{0, 115, 255},
	{0, 139, 0},
	{0, 139, 102},
	{0, 139, 204},
	{0, 162, 51},
	{0, 162, 153},
	{0, 162, 255},
	{0, 185, 0},
	{0, 185, 102},
	{0, 185, 204},
	{0, 208, 51},
	{0, 208, 153},
	{0, 208, 255},
	{0, 231, 0},
	{0, 231, 102},
	{0, 231, 204},
	{0, 255, 51},
	{0, 255, 153},
	{0, 255, 255},
	{42, 0, 51},
	{42, 0, 153},
	{42, 0, 255},
	{42, 23, 0},
	{42, 23, 102},
	{42, 23, 204},
	{42, 46, 51},
	{42, 46, 153},
	{42, 46, 255},
	{42, 69, 0},
	{42, 69, 102},
	{42, 69, 204},
	{42, 92, 51},
	{42, 92, 153},
	{42, 92, 255},
	{42, 115, 0},
	{42, 115, 102},
	{42, 115, 204},
	{42, 139, 51},
	{42, 139, 153},
	{42, 139, 255},
	{42, 162, 0},
	{42, 162, 102},
	{42, 162, 204},
	{42, 185, 51},
	{42, 185, 153},
	{42, 185, 255},
	{42, 208, 0},
	{42, 208, 102},
	{42, 208, 204},
	{42, 231, 51},
	{42, 231, 153},
	{42, 231, 255},
	{42, 255, 0},
	{42, 255, 102},
	{42, 255, 204},
	{85, 0, 0},
	{85, 0, 102},
	{85, 0, 204},
	{85, 23, 51},
	{85, 23, 153},
	{85, 23, 255},
	{85, 46, 0},
	{85, 46, 102},
	{85, 46, 204},
	{85, 69, 51},
	{85, 69, 153},
	{85, 69, 255},
	{85, 92, 0},
	{85, 92, 102},
	{85, 92, 204},
	{85, 115, 51},
	{85, 115, 153},
	{85, 115, 255},
	{85, 139, 0},
	{85, 139, 102},
	{85, 139, 204},
	{85, 162, 51},
	{85, 162, 153},
	{85, 162, 255},
	{85, 185, 0},
	{85, 185, 102},
	{85, 185, 204},
	{85, 208, 51},
	{85, 208, 153},
	{85, 208, 255},
	{85, 231, 0},
	{85, 231, 102},
	{85, 231, 204},
	{85, 255, 51},
	{85, 255, 153},
	{85, 255, 255},
	{127, 0, 51},
	{127, 0, 153},
	{127, 0, 255},
	{127, 23, 0},
	{127, 23, 102},
	{127, 23, 204},
	{127, 46, 51},
	{127, 46, 153},
	{127, 46, 255},
	{127, 69, 0},
	{127, 69, 102},
	{127, 69, 204},
	{127, 92, 51},
	{127, 92, 153},
	{127, 92, 255},
	{127, 115, 0},
	{127, 115, 102},
	{127, 115, 204},
	{127, 139, 51},
	{127, 139, 153},
	{127, 139, 255},
	{127, 162, 0},
	{127, 162, 102},
	{127, 162, 204},
	{127, 185, 51},
	{127, 185, 153},
	{127, 185, 255},
	{127, 208, 0},
	{127, 208, 102},
	{127, 208, 204},
	{127, 231, 51},
	{127, 231, 153},
	{127, 231, 255},
	{127, 255, 0},
	{127, 255, 102},
	{127, 255, 204},
	{170, 0, 0},
	{170, 0, 102},
	{170, 0, 204},
	{170, 23, 51},
	{170, 23, 153},
	{170, 23, 255},
	{170, 46, 0},
	{170, 46, 102},
	{170, 46, 204},
	{170, 69, 51},
	{170, 69, 153},
	{170, 69, 255},
	{170, 92, 0},
	{170, 92, 102},
	{170, 92, 204},
	{170, 115, 51},
	{170, 115, 153},
	{170, 115, 255},
	{170, 139, 0},
	{170, 139, 102},
	{170, 139, 204},
	{170, 162, 51},
	{170, 162, 153},
	{170, 162, 255},
	{170, 185, 0},
	{170, 185, 102},
	{170, 185, 204},
	{170, 208, 51},
	{170, 208, 153},
	{170, 208, 255},
	{170, 231, 0},
	{170, 231, 102},
	{170, 231, 204},
	{170, 255, 51},
	{170, 255, 153},
	{170, 255, 255},
	{212, 0, 51},
	{212, 0, 153},
	{212, 0, 255},
	{212, 23, 0},
	{212, 23, 102},
	{212, 23, 204},
	{212, 46, 51},
	{212, 46, 153},
	{212, 46, 255},
	{212, 69, 0},
	{212, 69, 102},
	{212, 69, 204},
	{212, 92, 51},
	{212, 92, 153},
	{212, 92, 255},
	{212, 115, 0},
	{212, 115, 102},
	{212, 115, 204},
	{212, 139, 51},
	{212, 139, 153},
	{212, 139, 255},
	{212, 162, 0},
	{212, 162, 102},
	{212, 162, 204},
	{212, 185, 51},
	{212, 185, 153},
	{212, 185, 255},
	{212, 208, 0},
	{212, 208, 102},
	{212, 208, 204},
	{212, 231, 51},
	{212, 231, 153},
	{212, 231, 255},
	{212, 255, 0},
	{212, 255, 102},
	{212, 255, 204},
	{255, 0, 0},
	{255, 0, 102},
	{255, 0, 204},
	{255, 23, 51},
	{255, 23, 153},
	{255, 23, 255},
	{255, 46, 0},
	{255, 46, 102},
	{255, 46, 204},
	{255, 69, 51},
	{255, 69, 153},
	{255, 69, 255},
	{255, 92, 0},
	{255, 92, 102},
	{255, 92, 204},
	{255, 115, 51},
	{255, 115, 153},
	{255, 115, 255},
	{255, 139, 0},
	{255, 139, 102},
	{255, 139, 204},
	{255, 162, 51},
	{255, 162, 153},
	{255, 162, 255},
	{255, 185, 0},
	{255, 185, 102},
	{255, 185, 204},
	{255, 208, 51},
	{255, 208, 153},
	{255, 208, 255},
	{255, 231, 0},
	{255, 231, 102},
	{255, 231, 204},
	{255, 255, 51},
	{255, 255, 153},
	{255, 255, 255},
	{204, 204, 204},
	{153, 153, 153},
	{102, 102, 102},
	{51, 51, 51},
}

// GeneralPalette can return a pretty general 256 color palette
func GeneralPalette() (pal color.Palette) {
	for _, rgb := range GeneralPalette256 {
		pal = append(pal, color.NRGBA{rgb[0], rgb[1], rgb[2], 255})
	}
	return pal
}
