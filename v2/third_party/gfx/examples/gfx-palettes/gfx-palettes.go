package main

import "github.com/peterhellberg/gfx"

func main() {
	for size, paletteLookup := range gfx.PalettesByNumberOfColors {
		for name, palette := range paletteLookup {
			dst := gfx.NewImage(size, 1)

			for x, c := range palette {
				dst.Set(x, 0, c)
			}

			filename := gfx.Sprintf("gfx-Palette%s.png", name)

			gfx.SavePNG(filename, gfx.NewResizedImage(dst, 1120, 96))
		}
	}
}
