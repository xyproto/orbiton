package main

import "github.com/peterhellberg/gfx"

func main() {
	sn := gfx.NewSimplexNoise(17)

	dst := gfx.NewImage(1024, 256)

	gfx.EachImageVec(dst, gfx.ZV, func(u gfx.Vec) {
		n := sn.Noise2D(u.X/900, u.Y/900)
		c := gfx.PaletteSplendor128.At(n / 2)

		gfx.SetVec(dst, u, c)
	})

	gfx.SavePNG("gfx-example-simplex.png", dst)
}
