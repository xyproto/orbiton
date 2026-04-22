package gfx

import "image"

// Tiles is a slice of paletted images.
type Tiles []PalettedImage

// Tileset is a paletted tileset.
type Tileset struct {
	Palette Palette     // Palette of the tileset.
	Size    image.Point // Size is the size of each tile.
	Tiles   Tiles       // Images contains all of the images in the tileset.
}

// TilesetData is the raw data in a tileset
type TilesetData [][]uint8

// NewTileset creates a new paletted tileset.
func NewTileset(p Palette, s image.Point, td TilesetData) *Tileset {
	ts := &Tileset{Palette: p, Size: s}

	for i := 0; i < len(td); i++ {
		ts.Tiles = append(ts.Tiles, NewTile(p, s.X, td[i]))
	}

	return ts
}

// NewTilesetFromImage creates a new paletted tileset based on the provided palette, tile size and image.
func NewTilesetFromImage(p Palette, tileSize image.Point, src image.Image) *Tileset {
	cols := src.Bounds().Dx() / tileSize.X
	rows := src.Bounds().Dy() / tileSize.Y

	tiles := make(Tiles, cols*rows)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			t := NewPaletted(tileSize.X, tileSize.Y, p)

			DrawSrc(t, t.Bounds(), src, Pt(col*tileSize.X, row*tileSize.Y))

			i := (row * cols) + col

			tiles[i] = t
		}
	}

	return &Tileset{Palette: p, Size: tileSize, Tiles: tiles}
}

// NewTile returns a new paletted image with the given pix, stride and palette.
func NewTile(p Palette, cols int, pix []uint8) *Paletted {
	return &Paletted{
		Palette: p,
		Stride:  cols,
		Pix:     pix,
		Rect:    calcRect(cols, pix),
	}
}

func calcRect(cols int, pix []uint8) image.Rectangle {
	s := calcSize(cols, pix)

	return IR(0, 0, s.X, s.Y)
}

func calcSize(cols int, pix []uint8) image.Point {
	l := len(pix)

	if l < cols {
		return Pt(cols, 1)
	}

	rows := l / cols

	if rows*cols == l {
		return Pt(cols, rows)
	}

	if rows%cols > 0 {
		rows++
	}

	return Pt(cols, rows)
}
