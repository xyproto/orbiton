package gfx

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"sort"
)

// Palette is a slice of colors.
type Palette []color.NRGBA

// Color returns the color at index n.
func (p Palette) Color(n int) color.NRGBA {
	if n >= 0 && n < p.Len() {
		return p[n]
	}

	return color.NRGBA{}
}

// Len returns the number of colors in the palette.
func (p Palette) Len() int {
	return len(p)
}

// Sort palette.
func (p Palette) Sort(less func(i, j int) bool) {
	sort.Slice(p, less)
}

// Random color from the palette.
func (p Palette) Random() color.NRGBA {
	return p[rand.Intn(p.Len())]
}

// Tile returns a new image based on the input image, but with colors from the palette.
func (p Palette) Tile(src image.Image) *Paletted {
	dst := NewPalettedImage(src.Bounds(), p)

	draw.Draw(dst, dst.Bounds(), src, image.ZP, draw.Src)

	return dst
}

// Convert returns the palette color closest to c in Euclidean R,G,B space.
func (p Palette) Convert(c color.Color) color.Color {
	if len(p) == 0 {
		return color.RGBA{}
	}

	return p[p.Index(c)]
}

// Index returns the index of the palette color closest to c in Euclidean
// R,G,B,A space.
func (p Palette) Index(c color.Color) int {
	cr, cg, cb, ca := c.RGBA()
	ret, bestSum := 0, uint32(1<<32-1)

	for i, v := range p {
		vr, vg, vb, va := v.RGBA()
		sum := sqDiff(cr, vr) + sqDiff(cg, vg) + sqDiff(cb, vb) + sqDiff(ca, va)

		if sum < bestSum {
			if sum == 0 {
				return i
			}

			ret, bestSum = i, sum
		}
	}

	return ret
}

// AsColorPalette converts the Palette to a color.Palette.
func (p Palette) AsColorPalette() color.Palette {
	var cp = make(color.Palette, len(p))

	for i, c := range p {
		cp[i] = c
	}

	return cp
}

// At returns the color at the given float64 value (range 0-1)
func (p Palette) At(t float64) color.Color {
	n := len(p)
	if t <= 0 || math.IsNaN(t) {
		return p[0]
	}
	if t >= 1 {
		return p[n-1]
	}

	i := int(math.Floor(t * float64(n-1)))
	s := 1 / float64(n-1)

	return LerpColors(p[i], p[i+1], (t-float64(i)*s)/s)
}

// sqDiff returns the squared-difference of x and y, shifted by 2 so that
// adding four of those won't overflow a uint32.
//
// x and y are both assumed to be in the range [0, 0xffff].
func sqDiff(x, y uint32) uint32 {
	d := x - y

	return (d * d) >> 2
}
