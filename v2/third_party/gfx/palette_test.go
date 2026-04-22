package gfx

import (
	"image/color"
	"testing"
)

func TestPaletteColor(t *testing.T) {
	c := PaletteEN4.Color(-1)

	if got, want := c.R, uint8(0); got != want {
		t.Fatalf("c.R = %d, want %d", got, want)
	}

	if got, want := c.G, uint8(0); got != want {
		t.Fatalf("c.G = %d, want %d", got, want)
	}

	if got, want := c.B, uint8(0); got != want {
		t.Fatalf("c.B = %d, want %d", got, want)
	}

	if got, want := c.A, uint8(0); got != want {
		t.Fatalf("c.A = %d, want %d", got, want)
	}
}

func TestPaletteLen(t *testing.T) {
	if got, want := PaletteEN4.Len(), 4; got != want {
		t.Fatalf("PaletteEN4.Len() = %d, want %d", got, want)
	}
}

func TestPaletteRandom(t *testing.T) {
	if c := PaletteEN4.Random(); c.R == 0 {
		t.Fatalf("unexpected color")
	}
}

func TestPaletteTile(t *testing.T) {
	src := NewImage(2, 2)

	src.Set(0, 0, ColorBlack)
	src.Set(1, 1, ColorWhite)
	src.Set(0, 1, ColorMagenta)

	pm := PaletteEN4.Tile(src)

	for _, tc := range []struct {
		x int
		y int
		i uint8
	}{
		{0, 0, 3},
		{1, 1, 0},
		{0, 1, 1},
	} {
		if got, want := pm.Index(tc.x, tc.y), tc.i; got != want {
			t.Fatalf("pm.Index(%d, %d) = %d, want %d", tc.x, tc.y, got, want)
		}
	}
}

func TestPaletteConvert(t *testing.T) {
	p := Palette{}

	if p.Convert(ColorMagenta).(color.RGBA).R != 0 {
		t.Fatalf("unexpected color")
	}

	c := PaletteEN4.Convert(ColorNRGBA(255, 0, 0, 255)).(color.NRGBA)

	if got, want := c.R, uint8(229); got != want {
		t.Fatalf("c.R = %d, want %d", got, want)
	}

	if got, want := c.G, uint8(176); got != want {
		t.Fatalf("c.G = %d, want %d", got, want)
	}

	if got, want := c.B, uint8(131); got != want {
		t.Fatalf("c.B = %d, want %d", got, want)
	}

	if got, want := c.A, uint8(255); got != want {
		t.Fatalf("c.A = %d, want %d", got, want)
	}
}

func TestPaletteAsColorPalette(t *testing.T) {
	p := PaletteEN4
	cp := p.AsColorPalette()

	if got, want := len(cp), len(p); got != want {
		t.Fatalf("len(cp) = %d, want %d", got, want)
	}
}

func TestPaletteCmplxPhaseAt(t *testing.T) {
	r, g, b, a := PaletteEN4.CmplxPhaseAt(complex(1, 5)).RGBA()

	if got, want := r, uint32(30011); got != want {
		t.Fatalf("r = %d, want %d", got, want)
	}

	if got, want := g, uint32(33553); got != want {
		t.Fatalf("g = %d, want %d", got, want)
	}

	if got, want := b, uint32(26943); got != want {
		t.Fatalf("b = %d, want %d", got, want)
	}

	if got, want := a, uint32(65535); got != want {
		t.Fatalf("a = %d, want %d", got, want)
	}
}

func TestPaletteAt(t *testing.T) {
	for _, tc := range []struct {
		p    Palette
		t    float64
		want color.Color
	}{
		{PaletteEN4, 2, PaletteEN4[3]},
		{PaletteEN4, 1, PaletteEN4[3]},
		{PaletteEN4, 0, PaletteEN4[0]},
		{PaletteEN4, -1, PaletteEN4[0]},
		{PaletteEN4, 0.5, color.RGBA64{37907, 36751, 28784, 65535}},
	} {
		r, g, b, a := tc.p.At(tc.t).RGBA()
		wr, wg, wb, wa := tc.want.RGBA()

		if got, want := r, wr; got != want {
			t.Fatalf("r = %d, want %d", got, want)
		}

		if got, want := g, wg; got != want {
			t.Fatalf("g = %d, want %d", got, want)
		}

		if got, want := b, wb; got != want {
			t.Fatalf("b = %d, want %d", got, want)
		}

		if got, want := a, wa; got != want {
			t.Fatalf("a = %d, want %d", got, want)
		}
	}
}
