package main

import "github.com/peterhellberg/gfx"

func main() {
	c := gfx.PaletteEDG36.Color
	m := gfx.NewImage(1024, 256, c(5))

	gfx.EachPixel(m.Bounds(), func(x, y int) {
		sd := gfx.SignedDistance{gfx.IV(x, y)}

		if d := sd.OpRepeat(gfx.V(128, 128), func(sd gfx.SignedDistance) float64 {
			return sd.OpSubtraction(sd.Circle(50), sd.Line(gfx.V(0, 0), gfx.V(64, 64)))
		}); d < 40 {
			m.Set(x, y, c(int(gfx.MathAbs(d/5))))
		}
	})

	gfx.SavePNG("gfx-example-sdf.png", m)
}
