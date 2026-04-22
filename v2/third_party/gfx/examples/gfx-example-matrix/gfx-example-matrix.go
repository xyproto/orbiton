package main

import "github.com/peterhellberg/gfx"

var en4 = gfx.PaletteEN4

func main() {
	a := &gfx.Animation{Delay: 10}

	c := gfx.V(128, 128)

	p := gfx.Polygon{
		{50, 50},
		{50, 206},
		{128, 96},
		{206, 206},
		{206, 50},
	}

	for d := 0.0; d < 360; d += 2 {
		m := gfx.NewPaletted(256, 256, en4, en4.Color(3))

		matrix := gfx.IM.RotatedDegrees(c, d)

		gfx.DrawPolygon(m, p.Project(matrix), 0, en4.Color(2))
		gfx.DrawPolygon(m, p.Project(matrix.Scaled(c, 0.5)), 0, en4.Color(1))

		gfx.DrawCircleFilled(m, c, 5, en4.Color(0))

		a.AddPalettedImage(m)
	}

	a.SaveGIF("/tmp/gfx-readme-examples-matrix.gif")
}
