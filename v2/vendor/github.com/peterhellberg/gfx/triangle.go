package gfx

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// Triangle is an array of three vertexes
type Triangle [3]Vertex

// NewTriangle creates a new triangle.
func NewTriangle(i int, td *TrianglesData) Triangle {
	var t Triangle

	t[0].Position = td.Position(i)
	t[1].Position = td.Position(i + 1)
	t[2].Position = td.Position(i + 2)

	t[0].Color = td.Color(i)
	t[1].Color = td.Color(i + 1)
	t[2].Color = td.Color(i + 2)

	return t
}

// T constructs a new triangle based on three vertexes.
func T(a, b, c Vertex) Triangle {
	return Triangle{a, b, c}
}

// Positions returns the three positions.
func (t Triangle) Positions() (Vec, Vec, Vec) {
	return t[0].Position, t[1].Position, t[2].Position
}

// Colors returns the three colors.
func (t Triangle) Colors() (color.NRGBA, color.NRGBA, color.NRGBA) {
	return t[0].Color, t[1].Color, t[2].Color
}

// Bounds returns the bounds of the triangle.
func (t Triangle) Bounds() image.Rectangle {
	return t.Rect().Bounds()
}

// Rect returns the triangle Rect.
func (t Triangle) Rect() Rect {
	a, b, c := t.Positions()

	return R(
		math.Min(a.X, math.Min(b.X, c.X)),
		math.Min(a.Y, math.Min(b.Y, c.Y)),
		math.Max(a.X, math.Max(b.X, c.X)),
		math.Max(a.Y, math.Max(b.Y, c.Y)),
	)
}

// Color returns the color at vector u.
func (t Triangle) Color(u Vec) color.Color {
	o := t.Centroid()

	if triangleContains(u, t[0].Position, t[1].Position, o) {
		return t[1].Color
	}

	if triangleContains(u, t[1].Position, t[2].Position, o) {
		return t[2].Color
	}

	return t[0].Color
}

// Contains returns true if the given vector is inside the triangle.
func (t Triangle) Contains(u Vec) bool {
	a, b, c := t.Positions()

	return triangleContains(u, a, b, c)
}

func triangleContains(u, a, b, c Vec) bool {
	vs1 := b.Sub(a)
	vs2 := c.Sub(a)

	q := u.Sub(a)

	bs := q.Cross(vs2) / vs1.Cross(vs2)
	bt := vs1.Cross(q) / vs1.Cross(vs2)

	return bs >= 0 && bt >= 0 && bs+bt <= 1
}

// Centroid returns the centroid O of the triangle.
func (t Triangle) Centroid() Vec {
	a, b, c := t.Positions()

	return V(
		(a.X+b.X+c.X)/3,
		(a.Y+b.Y+c.Y)/3,
	)
}

// TriangleFunc is a function type that is called by Triangle.EachPixel
type TriangleFunc func(u Vec, t Triangle)

// EachPixel calls the given TriangleFunc for each pixel in the triangle.
func (t Triangle) EachPixel(tf TriangleFunc) {
	b := t.Bounds()

	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if u := IV(x, y); t.Contains(u) {
				tf(u, t)
			}
		}
	}
}

// Draw the triangle to dst.
func (t Triangle) Draw(dst draw.Image) (drawCount int) {
	b := t.Bounds()

	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if u := IV(x, y); t.Contains(u) {
				drawCount++
				SetVec(dst, u, t.Color(u))
			}
		}
	}

	return drawCount
}

// DrawOver draws the first color in the triangle over dst.
func (t Triangle) DrawOver(dst draw.Image) (drawCount int) {
	a, _, _ := t.Colors()

	return t.DrawColorOver(dst, a)
}

// DrawColor draws the triangle on dst using the given color.
func (t Triangle) DrawColor(dst draw.Image, c color.Color) (drawCount int) {
	return t.drawColor(dst, c, draw.Src)
}

// DrawColorOver draws the triangle over dst using the given color.
func (t Triangle) DrawColorOver(dst draw.Image, c color.Color) (drawCount int) {
	return t.drawColor(dst, c, draw.Over)
}

func (t Triangle) drawColor(dst draw.Image, c color.Color, op draw.Op) (drawCount int) {
	b := t.Bounds()

	var lefts, rights []Vec
	var invalid = V(-math.MaxInt64, 0)

	for y := b.Min.Y; y < b.Max.Y; y++ {
		var left, right = invalid, invalid

		for x := b.Min.X; x < b.Max.X; x++ {
			if u := IV(x, y); t.Contains(u) {
				left = u
				break
			}
		}

		for x := b.Max.X; x > b.Min.X; x-- {
			if u := IV(x, y); t.Contains(u) {
				right = u
				break
			}
		}

		if left != invalid && right != invalid {
			lefts = append(lefts, left)
			rights = append(rights, right)
		}
	}

	uc := NewUniform(c)

	for i := 0; i < len(lefts); i++ {
		r := NewRect(lefts[i], rights[i].AddXY(0, 1)).Bounds()

		draw.Draw(dst, r, uc, ZP, op)

		drawCount++
	}

	return drawCount
}

// DrawWireframe draws the triangle as a wireframe on dst.
func (t Triangle) DrawWireframe(dst draw.Image, c color.Color) (drawCount int) {
	if l := t[0].Position.To(t[1].Position).Len(); l > 25 {
		DrawLine(dst, t[0].Position, t[1].Position, l/50, ColorWithAlpha(c, 128))
		drawCount++
	}

	if l := t[1].Position.To(t[2].Position).Len(); l > 25 {
		DrawLine(dst, t[1].Position, t[2].Position, l/50, ColorWithAlpha(c, 128))
		drawCount++
	}

	if l := t[0].Position.To(t[2].Position).Len(); l > 25 {
		DrawLine(dst, t[0].Position, t[2].Position, l/50, ColorWithAlpha(c, 128))
		drawCount++
	}

	DrawLine(dst, t[0].Position, t[1].Position, 1, c)
	DrawLine(dst, t[1].Position, t[2].Position, 1, c)
	DrawLine(dst, t[0].Position, t[2].Position, 1, c)

	drawCount += 3

	return
}
