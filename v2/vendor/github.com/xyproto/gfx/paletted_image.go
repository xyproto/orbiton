package gfx

import (
	"image"
	"image/color"
	"image/draw"
)

// PalettedImage interface is implemented by *Paletted
type PalettedImage interface {
	GfxPalette() Palette
	ColorPalette() color.Palette
	NRGBAAt(int, int) color.NRGBA
	AlphaAt(int, int) uint8
	image.PalettedImage
}

// PalettedDrawImage interface is implemented by *Paletted
type PalettedDrawImage interface {
	SetColorIndex(int, int, uint8)
	PalettedImage
}

// Paletted is an in-memory image of uint8 indices into a given palette.
type Paletted struct {
	// Pix holds the image's pixels, as palette indices. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*1].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
	// Palette is the image's palette.
	Palette Palette
}

// NewPaletted returns a new paletted image with the given width, height and palette.
func NewPaletted(w, h int, p Palette, colors ...color.Color) *Paletted {
	m := NewPalettedImage(IR(0, 0, w, h), p)

	if len(colors) > 0 {
		DrawSrc(m, m.Bounds(), NewUniform(colors[0]), ZP)
	}

	return m
}

// NewPalettedImage returns a new paletted image with the given bounds and palette.
func NewPalettedImage(r image.Rectangle, p Palette) *Paletted {
	w, h := r.Dx(), r.Dy()

	pix := make([]uint8, 1*w*h)

	return &Paletted{
		Pix:     pix,
		Stride:  1 * w,
		Rect:    r,
		Palette: p,
	}
}

// NewResizedPalettedImage returns an image with the provided dimensions.
func NewResizedPalettedImage(src PalettedImage, w, h int) *Paletted {
	dst := NewPalettedImage(IR(0, 0, w, h), src.GfxPalette())

	ResizeImage(dst, src)

	return dst
}

// NewScaledPalettedImage returns a paletted image scaled by the provided scaling factor.
func NewScaledPalettedImage(src PalettedImage, s float64) *Paletted {
	b := src.Bounds()

	return NewResizedPalettedImage(src, int(float64(b.Dx())*s), int(float64(b.Dy())*s))
}

// ColorModel returns the color model of the paletted image.
func (p *Paletted) ColorModel() color.Model {
	return p.Palette
}

// Bounds returns the bounds of the paletted image.
func (p *Paletted) Bounds() image.Rectangle {
	return p.Rect
}

// GfxPalette returns the gfx palette of the paletted image.
func (p *Paletted) GfxPalette() Palette {
	return p.Palette
}

// ColorPalette returns the color palette of the paletted image.
func (p *Paletted) ColorPalette() color.Palette {
	return p.Palette.AsColorPalette()
}

// At returns the color at (x, y).
func (p *Paletted) At(x, y int) color.Color {
	return p.NRGBAAt(x, y)
}

// NRGBAAt returns the color.NRGBA at (x, y).
func (p *Paletted) NRGBAAt(x, y int) color.NRGBA {
	i := p.ColorIndexAt(x, y)

	if int(i) >= len(p.Palette) {
		return p.Palette[0]
	}

	return p.Palette[i]
}

// AlphaAt returns the alpha value at (x, y).
func (p *Paletted) AlphaAt(x, y int) uint8 {
	return p.Palette[p.ColorIndexAt(x, y)].A
}

// Pixels returns the pixels of the paletted image as a []uint8.
func (p *Paletted) Pixels() []uint8 {
	pix := make([]uint8, len(p.Pix)*4)

	for i, n := range p.Pix {
		o := i * 4
		c := p.Palette[int(n)]
		pix[o], pix[o+1], pix[o+2], pix[o+3] = c.R, c.G, c.B, c.A
	}

	return pix
}

// PixOffset returns the index of the first element of Pix
// that corresponds to the pixel at (x, y).
func (p *Paletted) PixOffset(x, y int) int {
	return y*p.Stride + x
	//return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*1
}

// Set changes the color at (x, y).
func (p *Paletted) Set(x, y int, c color.Color) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}

	i := p.PixOffset(x, y)

	p.Pix[i] = uint8(p.Palette.Index(c))
}

// Index returns the color index at (x, y). (Short for ColorIndexAt)
func (p *Paletted) Index(x, y int) uint8 {
	return p.ColorIndexAt(x, y)
}

// Put changes the color index at (x, y). (Short for SetColorIndex)
func (p *Paletted) Put(x, y int, index uint8) {
	p.SetColorIndex(x, y, index)
}

// ColorIndexAt returns the color index at (x, y).
func (p *Paletted) ColorIndexAt(x, y int) uint8 {
	if o := p.PixOffset(x, y); o < len(p.Pix) {
		return p.Pix[o]
	}

	return 0
}

// SetColorIndex changes the color index at (x, y).
func (p *Paletted) SetColorIndex(x, y int, index uint8) {
	p.Pix[p.PixOffset(x, y)] = index
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *Paletted) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(p.Rect)

	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &Paletted{
			Palette: p.Palette,
		}
	}

	i := p.PixOffset(r.Min.X, r.Min.Y)

	return &Paletted{
		Pix:     p.Pix[i:],
		Stride:  p.Stride,
		Rect:    p.Rect.Intersect(r),
		Palette: p.Palette,
	}
}

// Opaque scans the entire image and reports whether it is fully opaque.
func (p *Paletted) Opaque() bool {
	var present [256]bool

	i0, i1 := 0, p.Rect.Dx()

	for y := p.Rect.Min.Y; y < p.Rect.Max.Y; y++ {
		for _, c := range p.Pix[i0:i1] {
			present[c] = true
		}

		i0 += p.Stride
		i1 += p.Stride
	}

	for i, c := range p.Palette {
		if !present[i] {
			continue
		}

		_, _, _, a := c.RGBA()

		if a != 0xffff {
			return false
		}
	}

	return true
}

// Make sure that *PalettedImage implements these interfaces
var (
	_ PalettedImage = &Paletted{}
	_ draw.Image    = &Paletted{}
)
