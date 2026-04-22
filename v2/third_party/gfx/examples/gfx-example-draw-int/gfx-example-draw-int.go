package main

import "github.com/peterhellberg/gfx"

func main() {
	m := gfx.NewImage(160, 128, gfx.ColorTransparent)

	p := gfx.PaletteNight16

	gfx.DrawIntLine(m, 10, 10, 94, 10, p.Color(0))
	gfx.DrawIntLine(m, 94, 16, 10, 16, p.Color(1))
	gfx.DrawIntLine(m, 10, 20, 10, 118, p.Color(2))
	gfx.DrawIntLine(m, 16, 118, 16, 20, p.Color(4))

	gfx.DrawIntLine(m, 40, 40, 80, 80, p.Color(5))
	gfx.DrawIntLine(m, 40, 40, 80, 70, p.Color(6))
	gfx.DrawIntLine(m, 40, 40, 80, 60, p.Color(7))
	gfx.DrawIntLine(m, 40, 40, 80, 50, p.Color(8))
	gfx.DrawIntLine(m, 40, 40, 80, 40, p.Color(9))

	gfx.DrawIntLine(m, 100, 100, 40, 100, p.Color(10))
	gfx.DrawIntLine(m, 100, 100, 40, 90, p.Color(11))
	gfx.DrawIntLine(m, 100, 100, 40, 80, p.Color(12))
	gfx.DrawIntLine(m, 100, 100, 40, 70, p.Color(13))
	gfx.DrawIntLine(m, 100, 100, 40, 60, p.Color(14))
	gfx.DrawIntLine(m, 100, 100, 40, 50, p.Color(15))

	gfx.DrawIntRectangle(m, 30, 106, 120, 20, p.Color(14))
	gfx.DrawIntFilledRectangle(m, 34, 110, 112, 12, p.Color(8))

	gfx.DrawIntCircle(m, 120, 30, 20, p.Color(5))
	gfx.DrawIntFilledCircle(m, 120, 30, 16, p.Color(4))

	gfx.DrawIntTriangle(m, 120, 102, 100, 80, 152, 46, p.Color(9))
	gfx.DrawIntFilledTriangle(m, 119, 98, 105, 80, 144, 54, p.Color(6))

	s := gfx.NewScaledImage(m, 6)

	gfx.SavePNG("gfx-example-draw-int.png", s)
}
