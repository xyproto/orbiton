package util

import (
	"io"
)

// COULD try out the new Go 1.23 iter package, but to keep backwards compatibility will
// just use something basic and simple.
func RangeIterator(startX uint32, startY uint32, endX uint32, endY uint32) func() (*Point, error) {
	x := startX
	y := startY
	return func() (*Point, error) {
		if x > endX {
			x = startX
			y++
		}
		if y > endY {
			return nil, io.EOF
		}
		x++
		return &Point{X: int32(x), Y: int32(y)}, nil
	}
}

func RangeIteratorWithIntPoint(ip Point) func() (*Point, error) {
	return RangeIterator(0, 0, uint32(ip.X), uint32(ip.Y))
}
