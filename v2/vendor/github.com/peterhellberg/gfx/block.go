//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image/draw"
	"math"
)

// Block has a position, size and color.
type Block struct {
	Pos   Vec3
	Size  Vec3
	Color BlockColor
}

// NewBlock creates a new Block.
func NewBlock(pos, size Vec3, ic BlockColor) Block {
	return Block{Pos: pos, Size: size, Color: ic}
}

// Box creates a box for the Block.
func (b Block) Box() Box {
	return NewBox(b.Pos, b.Pos.Add(b.Size))
}

// Behind checks if the box for b is in front of the box for a.
func (b Block) Behind(a Block) bool {
	return b.Box().Behind(a.Box())
}

// TrianglesData creates triangles data for the Block.
func (b Block) TrianglesData(origin Vec3) *TrianglesData {
	sv, c := b.Corners(origin), b.Color

	return &TrianglesData{
		// Shape
		Vx(sv.LeftUp, c.Medium),
		Vx(sv.LeftDown, c.Medium),
		Vx(sv.RightDown, c.Medium),
		Vx(sv.LeftUp, c.Medium),
		Vx(sv.RightUp, c.Medium),
		Vx(sv.RightDown, c.Medium),
		Vx(sv.BackUp, c.Light),
		Vx(sv.LeftUp, c.Light),
		Vx(sv.RightUp, c.Light),
		Vx(sv.LeftDown, c.Medium),
		Vx(sv.FrontDown, c.Medium),
		Vx(sv.RightDown, c.Medium),

		// Top
		Vx(sv.LeftUp, c.Light),
		Vx(sv.BackUp, c.Light),
		Vx(sv.FrontUp, c.Light),
		Vx(sv.FrontUp, c.Light),
		Vx(sv.BackUp, c.Light),
		Vx(sv.RightUp, c.Light),

		// Left
		Vx(sv.LeftUp, c.Dark),
		Vx(sv.LeftDown, c.Dark),
		Vx(sv.FrontDown, c.Dark),
		Vx(sv.FrontUp, c.Dark),
		Vx(sv.LeftUp, c.Dark),
		Vx(sv.FrontDown, c.Dark),

		// Right
		Vx(sv.FrontUp, c.Medium),
		Vx(sv.RightUp, c.Medium),
		Vx(sv.RightDown, c.Medium),
		Vx(sv.FrontUp, c.Medium),
		Vx(sv.FrontDown, c.Medium),
		Vx(sv.RightDown, c.Medium),
	}
}

// Triangles for the Block
func (b Block) Triangles(origin Vec3) []Triangle {
	c, l, m, d := b.Corners(origin), b.Color.Light, b.Color.Medium, b.Color.Dark

	return []Triangle{
		T(Vx(c.FrontUp, d), Vx(c.LeftDown), Vx(c.LeftUp)),
		T(Vx(c.FrontUp, d), Vx(c.FrontDown), Vx(c.LeftDown)),
		T(Vx(c.FrontUp, m), Vx(c.FrontDown), Vx(c.RightDown)),
		T(Vx(c.FrontUp, m), Vx(c.RightUp), Vx(c.RightDown)),
		T(Vx(c.FrontUp, l), Vx(c.BackUp), Vx(c.LeftUp)),
		T(Vx(c.FrontUp, l), Vx(c.BackUp), Vx(c.RightUp)),
	}
}

// Draw block on dst at origin.
func (b Block) Draw(dst draw.Image, origin Vec3) {
	DrawTriangles(dst, b.Triangles(origin))
}

// DrawOver draws block over dst at origin.
func (b Block) DrawOver(dst draw.Image, origin Vec3) {
	DrawTrianglesOver(dst, b.Triangles(origin))
}

// DrawPolygons for block on dst at origin.
func (b Block) DrawPolygons(dst draw.Image, origin Vec3) {
	shape, top, left, _ := b.Polygons(origin)

	shape.Fill(dst, b.Color.Medium)
	left.Fill(dst, b.Color.Dark)
	top.Fill(dst, b.Color.Light)
}

// DrawRectangles for block on dst at origin.
func (b Block) DrawRectangles(dst draw.Image, origin Vec3) {
	_, top, left, right := b.Polygons(origin)

	t := top.Bounds()
	l := left.Bounds()
	r := right.Bounds()

	tx6 := t.Dx() / 6
	ty2 := t.Dy() / 4

	DrawColor(dst, IR(l.Min.X, l.Min.Y, l.Max.X, l.Max.Y-ty2), b.Color.Dark)
	DrawColor(dst, IR(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y-ty2), b.Color.Medium)
	DrawColor(dst, IR(t.Min.X+tx6, t.Min.Y+ty2, t.Max.X-tx6, t.Max.Y), b.Color.Light)
}

// DrawBounds for block on dst at origin.
func (b Block) DrawBounds(dst draw.Image, origin Vec3) {
	r := b.Rect(origin)

	if r.Area() < 20 {
		DrawColor(dst, r.Bounds(), b.Color.Medium)
		return
	}

	rw6 := r.W() / 6

	DrawColor(dst, IR(int(r.Min.X), int(r.Min.Y), int(r.Max.X), int(r.Max.Y-rw6)), b.Color.Light)

	DrawLineBresenham(dst, r.Min, V(r.Max.X, r.Min.Y), b.Color.Medium)
	DrawLineBresenham(dst, r.Min, V(r.Min.X, r.Max.Y-rw6), b.Color.Dark)
}

// DrawWireframe block on dst at origin.
func (b Block) DrawWireframe(dst draw.Image, origin Vec3) {
	DrawTrianglesWireframe(dst, b.Triangles(origin))
}

// Polygons returns the shape, top, left and right polygons with coordinates based on origin.
func (b Block) Polygons(origin Vec3) (shape, top, left, right Polygon) {
	vs := b.Corners(origin)

	return vs.Shape(), vs.Top(), vs.Left(), vs.Right()
}

// Rect for the block.
func (b Block) Rect(origin Vec3) Rect {
	return b.Corners(origin).Rect()
}

// Shape returns the shape Polygon
func (b Block) Shape(origin Vec3) Polygon {
	return b.Corners(origin).Shape()
}

// Space returns the BlockSpace for the Block.
func (b Block) Space() BlockSpace {
	p, s := b.Pos, b.Size

	return BlockSpace{
		LeftUp:    V3(p.X, p.Y+s.Y, p.Z+s.Z),
		LeftDown:  V3(p.X, p.Y+s.Y, p.Z),
		FrontDown: V3(p.X, p.Y, p.Z),
		RightDown: V3(p.X+s.X, p.Y, p.Z),
		RightUp:   V3(p.X+s.X, p.Y, p.Z+s.Z),
		BackUp:    V3(p.X+s.X, p.Y+s.Y, p.Z+s.Z),
		BackDown:  V3(p.X+s.X, p.Y+s.Y, p.Z),
		FrontUp:   V3(p.X, p.Y, p.Z+s.Z),
	}
}

// Corners returns the screen coordinates for the Block corners.
func (b Block) Corners(origin Vec3) BlockCorners {
	return b.Space().Corners(origin)
}

// BlockCorners contains screen coordinates for all of the corners.
type BlockCorners struct {
	LeftUp    Vec
	LeftDown  Vec
	FrontDown Vec
	RightDown Vec
	RightUp   Vec
	BackUp    Vec
	BackDown  Vec
	FrontUp   Vec
}

// Rect for the Block corners.
func (bc BlockCorners) Rect() Rect {
	return R(bc.LeftUp.X, bc.BackUp.Y, bc.RightUp.X, bc.FrontDown.Y)
}

// Shape Polygon.
func (bc BlockCorners) Shape() Polygon {
	return Polygon{bc.LeftUp, bc.LeftDown, bc.FrontDown, bc.RightDown, bc.RightUp, bc.BackUp}
}

// Top face Polygon.
func (bc BlockCorners) Top() Polygon {
	return Polygon{bc.LeftUp, bc.FrontUp, bc.RightUp, bc.BackUp}
}

// Left face Polygon.
func (bc BlockCorners) Left() Polygon {
	return Polygon{bc.LeftUp, bc.LeftDown, bc.FrontDown, bc.FrontUp}
}

// Right face Polygon.
func (bc BlockCorners) Right() Polygon {
	return Polygon{bc.FrontUp, bc.FrontDown, bc.RightDown, bc.RightUp}
}

// BlockSpace contains 3D space coordinates for the block corners.
type BlockSpace struct {
	LeftUp    Vec3
	LeftDown  Vec3
	FrontDown Vec3
	RightDown Vec3
	RightUp   Vec3
	BackUp    Vec3
	BackDown  Vec3
	FrontUp   Vec3
}

// Corners returns the screen coordinates for all of the Block corners.
func (bs BlockSpace) Corners(origin Vec3) BlockCorners {
	return BlockCorners{
		LeftUp:    bs.CornerLeftUp(origin),
		LeftDown:  bs.CornerLeftDown(origin),
		FrontDown: bs.CornerFrontDown(origin),
		RightDown: bs.CornerRightDown(origin),
		RightUp:   bs.CornerRightUp(origin),
		BackUp:    bs.CornerBackUp(origin),
		BackDown:  bs.CornerBackDown(origin),
		FrontUp:   bs.CornerFrontUp(origin),
	}
}

// CornerLeftUp returns the screen coordinate for the LeftUp corner.
func (bs BlockSpace) CornerLeftUp(origin Vec3) Vec {
	return blockCorner(bs.LeftUp, origin)
}

// CornerLeftDown returns the screen coordinate for the LeftDown corner.
func (bs BlockSpace) CornerLeftDown(origin Vec3) Vec {
	return blockCorner(bs.LeftDown, origin)
}

// CornerFrontDown returns the screen coordinate for the FrontDown corner.
func (bs BlockSpace) CornerFrontDown(origin Vec3) Vec {
	return blockCorner(bs.FrontDown, origin)
}

// CornerRightDown returns the screen coordinate for the RightDown corner.
func (bs BlockSpace) CornerRightDown(origin Vec3) Vec {
	return blockCorner(bs.RightDown, origin)
}

// CornerRightUp returns the screen coordinate for the RightUp corner.
func (bs BlockSpace) CornerRightUp(origin Vec3) Vec {
	return blockCorner(bs.RightUp, origin)
}

// CornerBackUp returns the screen coordinate for the BackUp corner.
func (bs BlockSpace) CornerBackUp(origin Vec3) Vec {
	return blockCorner(bs.BackUp, origin)
}

// CornerBackDown returns the screen coordinate for the BackDown corner.
func (bs BlockSpace) CornerBackDown(origin Vec3) Vec {
	return blockCorner(bs.BackDown, origin)
}

// CornerFrontUp returns the screen coordinate for the FrontUp corner.
func (bs BlockSpace) CornerFrontUp(origin Vec3) Vec {
	return blockCorner(bs.FrontUp, origin)
}

// blockCorner converts a 3D space corner and origin into a screen coordinate corner.
func blockCorner(pos, origin Vec3) Vec {
	h, v := spaceToIso(pos)

	// Convert the given 2D isometric coordinates to 2D screen coordinates.
	x := h*origin.Z + origin.X
	y := -(v*origin.Z + origin.Y)

	return V(x, y)
}

// Convert 3D space coordinates to flattened 2D isometric coordinates.
// x and y coordinates are oblique axes separated by 120 degrees.
// h,v are the horizontal and vertical distances from the origin.
func spaceToIso(space Vec3) (h, v float64) {
	x, y := space.X+space.Z, space.Y+space.Z

	h = (x - y) * math.Sqrt(3) / 2
	v = (x + y) / 2

	return h, v
}
