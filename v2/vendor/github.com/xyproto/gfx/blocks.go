//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image/draw"
	"sort"
)

// Blocks is a slice of blocks.
type Blocks []Block

// Add appends one or more blocks to the slice of Blocks.
func (blocks *Blocks) Add(bs ...Block) {
	if len(bs) > 0 {
		*blocks = append(*blocks, bs...)
	}
}

// AddNewBlock creates a new Block and appends it to the slice.
func (blocks *Blocks) AddNewBlock(pos, size Vec3, ic BlockColor) {
	blocks.Add(NewBlock(pos, size, ic))
}

// Draw all blocks.
func (blocks Blocks) Draw(dst draw.Image, origin Vec3) {
	for _, block := range blocks {
		if block.Rect(origin).Bounds().Overlaps(dst.Bounds()) {
			block.Draw(dst, origin)
		}
	}
}

// DrawPolygons draws all of the blocks on the dst image.
// (using the shape, top and left polygons at the given origin)
func (blocks Blocks) DrawPolygons(dst draw.Image, origin Vec3) {
	for _, block := range blocks {
		if block.Rect(origin).Bounds().Overlaps(dst.Bounds()) {
			block.DrawPolygons(dst, origin)
		}
	}
}

// DrawRectangles for all blocks.
func (blocks Blocks) DrawRectangles(dst draw.Image, origin Vec3) {
	for _, block := range blocks {
		if block.Rect(origin).Bounds().Overlaps(dst.Bounds()) {
			block.DrawRectangles(dst, origin)
		}
	}
}

// DrawBounds for all blocks.
func (blocks Blocks) DrawBounds(dst draw.Image, origin Vec3) {
	for _, block := range blocks {
		if block.Rect(origin).Bounds().Overlaps(dst.Bounds()) {
			block.DrawBounds(dst, origin)
		}
	}
}

// DrawWireframes for all blocks.
func (blocks Blocks) DrawWireframes(dst draw.Image, origin Vec3) {
	for _, block := range blocks {
		if block.Rect(origin).Bounds().Overlaps(dst.Bounds()) {
			block.DrawWireframe(dst, origin)
		}
	}
}

// Sort blocks to be drawn starting from max X, max Y and min Z.
func (blocks Blocks) Sort() {
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Behind(blocks[j])
	})
}
