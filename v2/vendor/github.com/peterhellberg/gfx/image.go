package gfx

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
)

// ZP is the zero image.Point.
var ZP = image.ZP

// NewImage creates an image of the given size (optionally filled with a color)
func NewImage(w, h int, colors ...color.Color) *image.RGBA {
	m := NewRGBA(IR(0, 0, w, h))

	if len(colors) > 0 {
		DrawColor(m, m.Bounds(), colors[0])
	}

	return m
}

// NewNRGBA returns a new NRGBA image with the given bounds.
func NewNRGBA(r image.Rectangle) *image.NRGBA {
	return image.NewNRGBA(r)
}

// NewRGBA returns a new RGBA image with the given bounds.
func NewRGBA(r image.Rectangle) *image.RGBA {
	return image.NewRGBA(r)
}

// NewGray returns a new Gray image with the given bounds.
func NewGray(r image.Rectangle) *image.Gray {
	return image.NewGray(r)
}

// NewGray16 returns a new Gray16 image with the given bounds.
// (For example useful for height maps)
func NewGray16(r image.Rectangle) *image.Gray16 {
	return image.NewGray16(r)
}

// NewUniform creates a new uniform image of the given color.
func NewUniform(c color.Color) *image.Uniform {
	return image.NewUniform(c)
}

// Pt returns an image.Point for the given x and y.
func Pt(x, y int) image.Point {
	return image.Pt(x, y)
}

// IR returns an image.Rectangle for the given input.
func IR(x0, y0, x1, y1 int) image.Rectangle {
	return image.Rect(x0, y0, x1, y1)
}

// Mix the current pixel color at x and y with the given color.
func Mix(m draw.Image, x, y int, c color.Color) {
	_, _, _, a := c.RGBA()

	switch a {
	case 0xFFFF:
		m.Set(x, y, c)
	default:
		DrawColorOver(m, IR(x, y, x+1, y+1), c)
	}
}

// MixPoint the current pixel color at the image.Point with the given color.
func MixPoint(dst draw.Image, p image.Point, c color.Color) {
	Mix(dst, p.X, p.Y, c)
}

// Set x and y to the given color.
func Set(dst draw.Image, x, y int, c color.Color) {
	dst.Set(x, y, c)
}

// SetPoint to the given color.
func SetPoint(dst draw.Image, p image.Point, c color.Color) {
	dst.Set(p.X, p.Y, c)
}

// SetVec to the given color.
func SetVec(dst draw.Image, u Vec, c color.Color) {
	pt := u.Pt()

	dst.Set(pt.X, pt.Y, c)
}

// EachImageVec calls the provided function for each Vec
// in the provided image in the given direction.
//
// gfx.V(1,1) to call the function on each pixel starting from the top left.
func EachImageVec(src image.Image, dir Vec, fn func(u Vec)) {
	BoundsToRect(src.Bounds()).EachVec(dir, fn)
}

// EachPixel calls the provided function for each pixel in the provided rectangle.
func EachPixel(r image.Rectangle, fn func(x, y int)) {
	for x := r.Min.X; x < r.Max.X; x++ {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			fn(x, y)
		}
	}
}

// EncodePNG encodes an image as PNG to the provided io.Writer.
func EncodePNG(w io.Writer, src image.Image) error {
	return png.Encode(w, src)
}

// DecodePNG decodes a PNG from the provided io.Reader.
func DecodePNG(r io.Reader) (image.Image, error) {
	return png.Decode(r)
}

// DecodePNGBytes decodes a PNG from the provided []byte.
func DecodePNGBytes(b []byte) (image.Image, error) {
	return DecodePNG(bytes.NewReader(b))
}

// DecodeImage decodes an image from the provided io.Reader.
func DecodeImage(r io.Reader) (image.Image, error) {
	m, _, err := image.Decode(r)

	return m, err
}

// DecodeImageBytes decodes an image from the provided []byte.
func DecodeImageBytes(b []byte) (image.Image, error) {
	return DecodeImage(bytes.NewReader(b))
}
