package main

import "github.com/peterhellberg/gfx"

var (
	red   = gfx.BlockColorRed.Medium
	green = gfx.BlockColorGreen.Medium
	blue  = gfx.BlockColorBlue.Medium
)

func main() {
	m := gfx.NewImage(32, 16, gfx.ColorTransparent)

	gfx.DrawLineBresenham(m, gfx.V(2, 2), gfx.V(2, 14), red)
	gfx.DrawLineBresenham(m, gfx.V(6, 2), gfx.V(32, 2), green)
	gfx.DrawLineBresenham(m, gfx.V(6, 6), gfx.V(30, 14), blue)

	s := gfx.NewScaledImage(m, 16)

	gfx.SavePNG("gfx-example-bresenham-line.png", s)
}
