package gfx

import "image/color"

// Standard colors transparent, opaque, black, white, red, green, blue, cyan, magenta, and yellow.
var (
	ColorTransparent = ColorNRGBA(0, 0, 0, 0)
	ColorOpaque      = ColorNRGBA(0xFF, 0xFF, 0xFF, 0xFF)
	ColorBlack       = Palette1Bit.Color(0)
	ColorWhite       = Palette1Bit.Color(1)
	ColorRed         = Palette3Bit.Color(1)
	ColorGreen       = Palette3Bit.Color(2)
	ColorBlue        = Palette3Bit.Color(3)
	ColorCyan        = Palette3Bit.Color(4)
	ColorMagenta     = Palette3Bit.Color(5)
	ColorYellow      = Palette3Bit.Color(6)

	// ColorByName is a map of all the default colors by name.
	ColorByName = map[string]color.NRGBA{
		"Transparent": ColorTransparent,
		"Opaque":      ColorOpaque,
		"Black":       ColorBlack,
		"White":       ColorWhite,
		"Red":         ColorRed,
		"Green":       ColorGreen,
		"Blue":        ColorBlue,
		"Cyan":        ColorCyan,
		"Magenta":     ColorMagenta,
		"Yellow":      ColorYellow,
	}
)

// BlockColor contains a Light, Medium and Dark color.
type BlockColor struct {
	Light  color.NRGBA
	Medium color.NRGBA
	Dark   color.NRGBA
}

// Block colors, each containing a Light, Medium and Dark color.
var (
	// Default block colors based on PaletteTango.
	BlockColorYellow = BlockColor{Light: PaletteTango[0], Medium: PaletteTango[1], Dark: PaletteTango[2]}
	BlockColorOrange = BlockColor{Light: PaletteTango[3], Medium: PaletteTango[4], Dark: PaletteTango[5]}
	BlockColorBrown  = BlockColor{Light: PaletteTango[6], Medium: PaletteTango[7], Dark: PaletteTango[8]}
	BlockColorGreen  = BlockColor{Light: PaletteTango[9], Medium: PaletteTango[10], Dark: PaletteTango[11]}
	BlockColorBlue   = BlockColor{Light: PaletteTango[12], Medium: PaletteTango[13], Dark: PaletteTango[14]}
	BlockColorPurple = BlockColor{Light: PaletteTango[15], Medium: PaletteTango[16], Dark: PaletteTango[17]}
	BlockColorRed    = BlockColor{Light: PaletteTango[18], Medium: PaletteTango[19], Dark: PaletteTango[20]}
	BlockColorWhite  = BlockColor{Light: PaletteTango[21], Medium: PaletteTango[22], Dark: PaletteTango[23]}
	BlockColorBlack  = BlockColor{Light: PaletteTango[24], Medium: PaletteTango[25], Dark: PaletteTango[26]}

	// BlockColors is a slice of the default block colors.
	BlockColors = []BlockColor{
		BlockColorYellow,
		BlockColorOrange,
		BlockColorBrown,
		BlockColorGreen,
		BlockColorBlue,
		BlockColorPurple,
		BlockColorRed,
		BlockColorWhite,
		BlockColorBlack,
	}

	// Block colors based on the Go color palette.
	BlockColorGoGopherBlue = BlockColor{Dark: PaletteGo[0], Medium: PaletteGo[2], Light: PaletteGo[4]}
	BlockColorGoLightBlue  = BlockColor{Dark: PaletteGo[9], Medium: PaletteGo[11], Light: PaletteGo[13]}
	BlockColorGoAqua       = BlockColor{Dark: PaletteGo[18], Medium: PaletteGo[20], Light: PaletteGo[22]}
	BlockColorGoFuchsia    = BlockColor{Dark: PaletteGo[27], Medium: PaletteGo[29], Light: PaletteGo[31]}
	BlockColorGoBlack      = BlockColor{Dark: PaletteGo[36], Medium: PaletteGo[38], Light: PaletteGo[40]}
	BlockColorGoYellow     = BlockColor{Dark: PaletteGo[45], Medium: PaletteGo[47], Light: PaletteGo[49]}

	// BlockColorsGo is a slice of block colors based on the Go color palette.
	BlockColorsGo = []BlockColor{
		BlockColorGoGopherBlue,
		BlockColorGoLightBlue,
		BlockColorGoAqua,
		BlockColorGoFuchsia,
		BlockColorGoBlack,
		BlockColorGoYellow,
	}

	// BlockColorByName is a map of block colors by name.
	BlockColorByName = map[string]BlockColor{
		// Default block colors.
		"Yellow": BlockColorYellow,
		"Orange": BlockColorOrange,
		"Brown":  BlockColorBrown,
		"Green":  BlockColorGreen,
		"Blue":   BlockColorBlue,
		"Purple": BlockColorPurple,
		"Red":    BlockColorRed,
		"White":  BlockColorWhite,
		"Black":  BlockColorBlack,

		// Go palette block colors.
		"GoGopherBlue": BlockColorGoGopherBlue,
		"GoLightBlue":  BlockColorGoLightBlue,
		"GoAqua":       BlockColorGoAqua,
		"GoFuchsia":    BlockColorGoFuchsia,
		"GoBlack":      BlockColorGoBlack,
		"GoYellow":     BlockColorGoYellow,
	}
)

// ColorWithAlpha creates a new color.RGBA based
// on the provided color.Color and alpha arguments.
func ColorWithAlpha(c color.Color, a uint8) color.NRGBA {
	nc := color.NRGBAModel.Convert(c).(color.NRGBA)

	nc.A = a

	return nc
}

// ColorRGBA constructs a color.RGBA.
func ColorRGBA(r, g, b, a uint8) color.RGBA {
	return color.RGBA{r, g, b, a}
}

// ColorNRGBA constructs a color.NRGBA.
func ColorNRGBA(r, g, b, a uint8) color.NRGBA {
	return color.NRGBA{r, g, b, a}
}

// ColorGray construcs a color.Gray.
func ColorGray(y uint8) color.Gray {
	return color.Gray{y}
}

// ColorGray16 construcs a color.Gray16.
func ColorGray16(y uint16) color.Gray16 {
	return color.Gray16{y}
}

// LerpColors performs linear interpolation between two colors.
func LerpColors(c0, c1 color.Color, t float64) color.Color {
	switch {
	case t <= 0:
		return c0
	case t >= 1:
		return c1
	}

	r0, g0, b0, a0 := c0.RGBA()
	r1, g1, b1, a1 := c1.RGBA()

	fr0, fg0, fb0, fa0 := float64(r0), float64(g0), float64(b0), float64(a0)
	fr1, fg1, fb1, fa1 := float64(r1), float64(g1), float64(b1), float64(a1)

	return color.RGBA64{
		uint16(Clamp(fr0+(fr1-fr0)*t, 0, 0xffff)),
		uint16(Clamp(fg0+(fg1-fg0)*t, 0, 0xffff)),
		uint16(Clamp(fb0+(fb1-fb0)*t, 0, 0xffff)),
		uint16(Clamp(fa0+(fa1-fa0)*t, 0, 0xffff)),
	}
}
