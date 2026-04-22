package main

import "github.com/peterhellberg/gfx"

var edg32 = gfx.PaletteEDG32

func main() {
	m := gfx.NewNRGBA(gfx.IR(0, 0, 1024, 256))
	p := gfx.Polygon{
		{80, 40},
		{440, 60},
		{700, 200},
		{250, 230},
		{310, 140},
	}

	p.EachPixel(m, func(x, y int) {
		pv := gfx.IV(x, y)
		l := pv.To(p.Rect().Center()).Len()

		gfx.Mix(m, x, y, edg32.Color(int(l/18)%32))
	})

	for n, v := range p {
		c := edg32.Color(n * 4)

		gfx.DrawCircle(m, v, 15, 8, gfx.ColorWithAlpha(c, 96))
		gfx.DrawCircle(m, v, 16, 1, c)
	}

	gfx.SavePNG("gfx-example-polygon.png", m)
}
