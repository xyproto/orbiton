package gfx

import (
	"math"
	"testing"
)

func TestNewIMDraw(t *testing.T) {
	imd := NewIMDraw(nil)

	if imd.matrix != IM {
		t.Fatalf("unexpected matrix")
	}
}

func TestIMDrawClear(t *testing.T) {
	imd := NewIMDraw(nil)

	imd.Clear()
}

func TestIMDrawPush(t *testing.T) {
	imd := NewIMDraw(nil)

	imd.Push(V(1, 2), V(3, 4))
}

func TestIMDrawLine(t *testing.T) {
	imd := NewIMDraw(nil)

	imd.EndShape = SharpEndShape

	imd.Line(0)

	imd.Push(V(1, 2))
	imd.Line(1)

	imd.Push(V(1, 2), V(1, 2), V(10, 5))
	imd.Line(1)

	imd.EndShape = RoundEndShape

	imd.Push(V(1, 2), V(3, 4))
	imd.Line(2)

	imd.Push(V(1, 2), V(3, 4), V(10, 5))
	imd.Line(3)
}

func TestIMDrawRectangle(t *testing.T) {
	imd := NewIMDraw(nil)

	imd.Push(V(1, 2))
	imd.Rectangle(0)

	imd.Push(V(1, 2), V(3, 4))
	imd.Rectangle(0)

	imd.Push(V(3, 3), V(7, 8))
	imd.Rectangle(1)

	imd.Push(V(1, 2))
	imd.Rectangle(1)

}

func TestIMDrawPolygon(t *testing.T) {
	imd := NewIMDraw(nil)

	imd.Push(V(1, 2), V(3, 4), V(1, 6))
	imd.Polygon(0)

	imd.Push(V(3, 3), V(7, 8), V(10, 2))
	imd.Polygon(1)
}

func TestIMDrawCircle(t *testing.T) {
	imd := NewIMDraw(nil)

	imd.Push(V(1, 2))
	imd.Circle(100, 0)

	imd.Push(V(8, 8))
	imd.Circle(50, 5)
}

func TestIMDrawCircleArc(t *testing.T) {
	imd := NewIMDraw(nil)

	imd.EndShape = RoundEndShape

	imd.Push(V(1, 2))
	imd.CircleArc(40, 0, 8*math.Pi, 0)

	imd.Push(V(8, 8))
	imd.CircleArc(40, 0, 8*math.Pi, 2)
}

func TestIMDrawEllipse(t *testing.T) {
	imd := NewIMDraw(nil)

	imd.Push(V(1, 2))
	imd.Ellipse(V(5, 10), 0)

	imd.Push(V(8, 8))
	imd.Ellipse(V(10, 5), 2)
}

func TestIMDrawEllipseArc(t *testing.T) {
	imd := NewIMDraw(nil)

	imd.EndShape = SharpEndShape

	imd.Push(V(1, 2))
	imd.EllipseArc(V(5, 10), 0, 8*math.Pi, 0)

	imd.Push(V(8, 8))
	imd.EllipseArc(V(10, 5), 2, 4*math.Pi, 2)
}
