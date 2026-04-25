//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image"
	"image/color"
	"image/draw"
)

// DrawTarget draws to a draw.Image, projected through a Matrix.
type DrawTarget struct {
	dst draw.Image
	mat Matrix
}

var _ BasicTarget = (*DrawTarget)(nil)

// NewDrawTarget creates a new draw target.
func NewDrawTarget(dst draw.Image) *DrawTarget {
	return &DrawTarget{
		dst: dst,
		mat: IM,
	}
}

// SetMatrix sets the matrix of the draw target.
func (dt *DrawTarget) SetMatrix(mat Matrix) {
	dt.mat = mat
}

// Bounds of the draw target.
func (dt *DrawTarget) Bounds() image.Rectangle {
	return dt.dst.Bounds()
}

// Center vector of the draw target.
func (dt *DrawTarget) Center() Vec {
	return BoundsCenter(dt.dst.Bounds())
}

// ColorModel of the draw target.
func (dt *DrawTarget) ColorModel() color.Model {
	return dt.dst.ColorModel()
}

// At retrieves the color at (x, y).
func (dt *DrawTarget) At(x, y int) color.Color {
	p := dt.mat.Project(IV(x, y)).Pt()

	return dt.dst.At(p.X, p.Y)
}

// Set the color at (x, y). (Projected through the draw target Matrix)
func (dt *DrawTarget) Set(x, y int, c color.Color) {
	p := dt.mat.Project(IV(x, y)).Pt()

	dt.dst.Set(p.X, p.Y, c)
}

// MakePicture creates a TargetPicture for the provided Picture.
func (dt *DrawTarget) MakePicture(pic Picture) TargetPicture {
	panic(Error("*DrawTarget: not implemented yet."))
}

// MakeTriangles creates TargetTriangles for the given Triangles
func (dt *DrawTarget) MakeTriangles(t Triangles) TargetTriangles {
	return &targetTriangles{Triangles: t, dt: dt}
}

type targetTriangles struct {
	Triangles
	dt *DrawTarget
}

func (tt *targetTriangles) Draw() {
	td := MakeTrianglesData(tt.Len())

	td.Update(tt.Triangles)

	for i := 0; i < td.Len(); i += 3 {
		t := NewTriangle(i, td)
		b := t.Bounds()

		for x := b.Min.X; x < b.Max.X; x++ {
			for y := b.Min.Y; y < b.Max.Y; y++ {

				if u := IV(x, y); t.Contains(u) {
					tt.dt.Set(x, y, t.Color(u))
				}
			}
		}
	}
}
