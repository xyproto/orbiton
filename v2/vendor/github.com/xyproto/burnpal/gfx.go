package burnpal

import (
	"github.com/peterhellberg/gfx"
	"image/color"
)

// GfxPalette returns the palette as a gfx.Palette
func GfxPalette() gfx.Palette {
	return gfx.Palette(Pal)
}

// ColorPalette returns the palette as a color.Palette
func ColorPalette() color.Palette {
	return gfx.Palette(Pal).AsColorPalette()
}
