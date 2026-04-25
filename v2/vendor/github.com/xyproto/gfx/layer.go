package gfx

import (
	"image"
	"image/color"
)

// Layer represents a layer of paletted tiles.
type Layer struct {
	Tileset *Tileset
	Width   int // Width of the layer in number of tiles.
	Data    LayerData
}

// LayerData is the data for a layer.
type LayerData []int

// Size returns the size of the layer data given the number of columns.
func (ld LayerData) Size(cols int) image.Point {
	l := len(ld)

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

// NewLayer creates a new layer.
func NewLayer(tileset *Tileset, width int, data LayerData) *Layer {
	return &Layer{Tileset: tileset, Width: width, Data: data}
}

// At returns the color at (x, y).
func (l *Layer) At(x, y int) color.Color {
	return l.NRGBAAt(x, y)
}

// NRGBAAt returns the color.RGBA at (x, y).
func (l *Layer) NRGBAAt(x, y int) color.NRGBA {
	if i := l.TileIndexAt(x, y); i > -1 {
		s := l.Tileset.Size

		return l.Tileset.Tiles[i].NRGBAAt(x%s.X, y%s.Y)
	}

	return ColorTransparent
}

// AlphaAt returns the alpha value at (x, y).
func (l *Layer) AlphaAt(x, y int) uint8 {
	if i := l.TileIndexAt(x, y); i > -1 {
		tx, ty := x%l.Tileset.Size.X, y%l.Tileset.Size.Y
		return l.Tileset.Tiles[i].AlphaAt(tx, ty)
	}

	return 0
}

// Bounds returns the bounds of the paletted layer.
func (l *Layer) Bounds() image.Rectangle {
	lpix := len(l.Data)

	switch {
	case l.Width < 1, lpix == 0,
		l.Tileset == nil,
		l.Tileset.Size.X < 1, l.Tileset.Size.Y < 1:
		return ZR
	case lpix < l.Width:
		return IR(0, 0, l.Width, 1)
	}

	s := l.Data.Size(l.Width)

	w := s.X * l.Tileset.Size.X
	h := s.Y * l.Tileset.Size.Y

	return IR(0, 0, w, h)
}

// ColorModel returns the color model for the paletted layer.
func (l *Layer) ColorModel() color.Model {
	return color.RGBAModel
}

// ColorIndexAt returns the palette index of the pixel at (x, y).
func (l *Layer) ColorIndexAt(x, y int) uint8 {
	if t := l.TileAt(x, y); t != nil {
		ts := l.Tileset.Size
		return t.ColorIndexAt(x%ts.X, y%ts.Y)
	}

	return 0
}

// TileAt returns the tile image at (x, y).
func (l *Layer) TileAt(x, y int) image.PalettedImage {
	if i := l.TileIndexAt(x, y); i >= 0 && i < len(l.Tileset.Tiles) {
		return l.Tileset.Tiles[i]
	}

	return nil
}

// TileSize returns the tileset tile size.
func (l *Layer) TileSize() image.Point {
	return l.Tileset.Size
}

// GfxPalette retrieves the layer palette.
func (l *Layer) GfxPalette() Palette {
	return l.Tileset.Palette
}

// ColorPalette retrieves the layer palette.
func (l *Layer) ColorPalette() color.Palette {
	return l.Tileset.Palette.AsColorPalette()
}

// Index returns the tile index at (x, y). (Short for TileIndexAt)
func (l *Layer) Index(x, y int) int {
	return l.TileIndexAt(x, y)
}

// TileIndexAt returns the tile index at (x, y).
func (l *Layer) TileIndexAt(x, y int) int {
	s := l.Tileset.Size
	o := y/s.Y*l.Width + x/s.X

	if o >= 0 && o < len(l.Data) {
		return l.Data[o]
	}

	return -1
}

// DataAt returns the data at (dx, dy).
func (l *Layer) DataAt(dx, dy int) int {
	return l.Data[l.dataOffset(dx, dy)]
}

// Put changes the tile index at (dx, dy). (Short for SetTileIndex)
func (l *Layer) Put(dx, dy, index int) {
	l.SetTileIndex(dx, dy, index)
}

// SetTileIndex changes the tile index at (dx, dy).
func (l *Layer) SetTileIndex(dx, dy, index int) {
	if o := l.dataOffset(dx, dy); o >= 0 && o < len(l.Data) {
		l.Data[o] = index
	}
}

func (l *Layer) dataOffset(dx, dy int) int {
	return dy*l.Width + dx
}
