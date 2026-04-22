package gfx

import (
	"image/color"
	"image/draw"
)

// DrawIntLine draws a line between two points
func DrawIntLine(dst draw.Image, x0, y0, x1, y1 int, c color.Color) {
	if x0 == x1 {
		if y0 > y1 {
			y0, y1 = y1, y0
		}

		for ; y0 <= y1; y0++ {
			dst.Set(x0, y0, c)
		}
	} else if y0 == y1 {
		if x0 > x1 {
			x0, x1 = x1, x0
		}

		for ; x0 <= x1; x0++ {
			dst.Set(x0, y0, c)
		}
	} else { // Bresenham
		dx := x1 - x0

		if dx < 0 {
			dx = -dx
		}

		dy := y1 - y0

		if dy < 0 {
			dy = -dy
		}

		steep := dy > dx

		if steep {
			x0, x1, y0, y1 = y0, y1, x0, x1
		}

		if x0 > x1 {
			x0, x1, y0, y1 = x1, x0, y1, y0
		}

		dx = x1 - x0
		dy = y1 - y0

		ystep := 1

		if dy < 0 {
			dy = -dy
			ystep = -1
		}

		err := dx / 2

		for ; x0 <= x1; x0++ {
			if steep {
				dst.Set(y0, x0, c)
			} else {
				dst.Set(x0, y0, c)
			}

			err -= dy
			if err < 0 {
				y0 += ystep
				err += dx
			}
		}
	}
}

// DrawIntRectangle draws a rectangle given a point, width and height
func DrawIntRectangle(dst draw.Image, x, y, w, h int, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}

	DrawIntLine(dst, x, y, x+w-1, y, c)
	DrawIntLine(dst, x, y, x, y+h-1, c)
	DrawIntLine(dst, x+w-1, y, x+w-1, y+h-1, c)
	DrawIntLine(dst, x, y+h-1, x+w-1, y+h-1, c)

	return
}

// DrawIntFilledRectangle draws a filled rectangle given a point, width and height
func DrawIntFilledRectangle(dst draw.Image, x, y, w, h int, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}

	for i := x; i < x+w; i++ {
		DrawIntLine(dst, i, y, i, y+h-1, c)
	}

	return
}

// DrawIntCircle draws a circle given a point and radius
func DrawIntCircle(dst draw.Image, x0, y0, r int, c color.Color) {
	f := 1 - r

	ddfx := 1
	ddfy := -2 * r

	x := 0
	y := r

	dst.Set(x0, y0+r, c)
	dst.Set(x0, y0-r, c)
	dst.Set(x0+r, y0, c)
	dst.Set(x0-r, y0, c)

	for x < y {
		if f >= 0 {
			y--
			ddfy += 2
			f += ddfy
		}

		x++
		ddfx += 2
		f += ddfx

		dst.Set(x0+x, y0+y, c)
		dst.Set(x0-x, y0+y, c)
		dst.Set(x0+x, y0-y, c)
		dst.Set(x0-x, y0-y, c)
		dst.Set(x0+y, y0+x, c)
		dst.Set(x0-y, y0+x, c)
		dst.Set(x0+y, y0-x, c)
		dst.Set(x0-y, y0-x, c)
	}
}

// DrawIntFilledCircle draws a filled circle given a point and radius
func DrawIntFilledCircle(dst draw.Image, x0, y0, r int, c color.Color) {
	f := 1 - r

	ddfx := 1
	ddfy := -2 * r

	x := 0
	y := r

	DrawIntLine(dst, x0, y0-r, x0, y0+r, c)

	for x < y {
		if f >= 0 {
			y--
			ddfy += 2
			f += ddfy
		}

		x++
		ddfx += 2
		f += ddfx

		DrawIntLine(dst, x0+x, y0-y, x0+x, y0+y, c)
		DrawIntLine(dst, x0+y, y0-x, x0+y, y0+x, c)
		DrawIntLine(dst, x0-x, y0-y, x0-x, y0+y, c)
		DrawIntLine(dst, x0-y, y0-x, x0-y, y0+x, c)
	}
}

// DrawIntTriangle draws a triangle given three points
func DrawIntTriangle(dst draw.Image, x0, y0, x1, y1, x2, y2 int, c color.Color) {
	DrawIntLine(dst, x0, y0, x1, y1, c)
	DrawIntLine(dst, x0, y0, x2, y2, c)
	DrawIntLine(dst, x1, y1, x2, y2, c)
}

// DrawIntFilledTriangle draws a filled triangle given three points
func DrawIntFilledTriangle(dst draw.Image, x0, y0, x1, y1, x2, y2 int, c color.Color) {
	if y0 > y1 {
		x0, y0, x1, y1 = x1, y1, x0, y0
	}

	if y1 > y2 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}

	if y0 > y1 {
		x0, y0, x1, y1 = x1, y1, x0, y0
	}

	if y0 == y2 {
		a := x0
		b := x0

		if x1 < a {
			a = x1
		} else if x1 > b {
			b = x1
		}

		if x2 < a {
			a = x2
		} else if x2 > b {
			b = x2
		}

		DrawIntLine(dst, a, y0, b, y0, c)

		return
	}

	dx01 := x1 - x0
	dy01 := y1 - y0
	dx02 := x2 - x0
	dy02 := y2 - y0
	dx12 := x2 - x1
	dy12 := y2 - y1

	sa := 0
	sb := 0
	a := 0
	b := 0

	last := y1 - 1

	if y1 == y2 {
		last = y1
	}

	for y := y0; y <= last; y++ {
		a = x0 + sa/dy01
		b = x0 + sb/dy02

		sa += dx01
		sb += dx02

		DrawIntLine(dst, a, y, b, y, c)
	}

	sa = dx12 * (last - y1)
	sb = dx02 * (last - y0)

	for y := last; y <= y2; y++ {
		a = x1 + sa/dy12
		b = x0 + sb/dy02

		sa += dx12
		sb += dx02

		DrawIntLine(dst, a, y, b, y, c)
	}
}
