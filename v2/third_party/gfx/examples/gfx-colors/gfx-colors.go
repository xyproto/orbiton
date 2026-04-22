package main

import "github.com/peterhellberg/gfx"

func main() {
	for name, c := range gfx.ColorByName {
		dst := gfx.NewImage(1, 1, c)
		filename := gfx.Sprintf("gfx-Color%s.png", name)

		gfx.SavePNG(filename, gfx.NewResizedImage(dst, 666, 48))
	}

	for name, bc := range gfx.BlockColorByName {
		dst := gfx.NewImage(3, 1)

		dst.Set(0, 0, bc.Dark)
		dst.Set(1, 0, bc.Medium)
		dst.Set(2, 0, bc.Light)

		filename := gfx.Sprintf("gfx-BlockColor%s.png", name)

		gfx.SavePNG(filename, gfx.NewResizedImage(dst, 619, 48))
	}
}
