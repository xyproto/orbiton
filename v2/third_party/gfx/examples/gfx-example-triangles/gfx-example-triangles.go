package main

import "github.com/peterhellberg/gfx"

var p = gfx.PaletteFamicube

func main() {
	n := 50
	m := gfx.NewPaletted(900, 270, p, p.Color(n+7))
	t := gfx.NewDrawTarget(m)

	t.MakeTriangles(&gfx.TrianglesData{
		vx(114, 16, n+1), vx(56, 142, n+2), vx(352, 142, n+3),
		vx(350, 142, n+4), vx(500, 50, n+5), vx(640, 236, n+6),
		vx(640, 70, n+8), vx(820, 160, n+9), vx(670, 236, n+10),
	}).Draw()

	gfx.SavePNG("gfx-example-triangles.png", m)
}

func vx(x, y float64, n int) gfx.Vertex {
	return gfx.Vertex{Position: gfx.V(x, y), Color: p.Color(n)}
}
