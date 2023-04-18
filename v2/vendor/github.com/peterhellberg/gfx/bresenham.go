package gfx

import (
	"image/color"
	"image/draw"
	"math"
)

// DrawLineBresenham draws a line using Bresenham's line algorithm.
//
// http://en.wikipedia.org/wiki/Bresenham's_line_algorithm
func DrawLineBresenham(dst draw.Image, from, to Vec, c color.Color) {
	x0, y0 := from.XY()
	x1, y1 := to.XY()

	steep := math.Abs(y0-y1) > math.Abs(x0-x1)

	if steep {
		x0, y0 = y0, x0
		x1, y1 = y1, x1
	}

	if x0 > x1 {
		x0, x1 = x1, x0
		y0, y1 = y1, y0
	}

	dx := x1 - x0
	dy := math.Abs(y1 - y0)
	e := dx / 2
	y := y0

	var ystep float64 = -1

	if y0 < y1 {
		ystep = 1
	}

	for x := x0; x <= x1; x++ {
		if steep {
			Mix(dst, int(y), int(x), c)
		} else {
			Mix(dst, int(x), int(y), c)
		}

		e -= dy

		if e < 0 {
			y += ystep
			e += dx
		}
	}
}
