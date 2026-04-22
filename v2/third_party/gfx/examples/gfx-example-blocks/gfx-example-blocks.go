package main

import "github.com/peterhellberg/gfx"

func main() {
	var (
		dst    = gfx.NewPaletted(898, 330, gfx.PaletteGo, gfx.PaletteGo[14])
		rect   = gfx.BoundsToRect(dst.Bounds())
		origin = rect.Center().ScaledXY(gfx.V(1.5, -2.5)).Vec3(0.55)
		blocks gfx.Blocks
	)

	for i, bc := range gfx.BlockColorsGo {
		var (
			f    = float64(i) + 0.5
			v    = f * 11
			pos  = gfx.V3(290+(v*3), 8.5*v, 9*(f+2))
			size = gfx.V3(90, 90, 90)
		)

		blocks.AddNewBlock(pos, size, bc)
	}

	blocks.Draw(dst, origin)

	gfx.SavePNG("gfx-example-blocks.png", dst)
}
