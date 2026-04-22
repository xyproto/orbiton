package gfx

import (
	"image"
	"image/color"
	"reflect"
	"testing"
)

func TestLayerData(t *testing.T) {
	for _, tc := range []struct {
		n    int
		data LayerData
		want image.Point
	}{
		{3, LayerData{}, Pt(3, 1)},
		{3, LayerData{0, 1}, Pt(3, 1)},
		{3, LayerData{0, 1, 2, 3}, Pt(3, 2)},
		{3, LayerData{0, 1, 2, 3, 4, 5, 6}, Pt(3, 3)},
		{4, LayerData{0, 1, 2, 3, 4, 5, 6}, Pt(4, 2)},
		{2, LayerData{0, 1, 2, 3, 4, 5, 6}, Pt(2, 4)},
		{3, LayerData{0, 1, 2, 3, 4, 5}, Pt(3, 2)},
	} {
		if got := tc.data.Size(tc.n); !got.Eq(tc.want) {
			t.Errorf("%v.Size(%d) = %v, want %v", tc.data, tc.n, got, tc.want)
		}
	}
}

func TestNewLayer(t *testing.T) {
	l := NewLayer(&Tileset{}, 123, LayerData{})

	if got, want := l.Width, 123; got != want {
		t.Fatalf("l.Width = %d, want %d", got, want)
	}
}

func TestLayerAt(t *testing.T) {
	l := newTestLayer()

	r, g, b, a := l.At(2, 2).RGBA()

	if got, want := r, uint32(16962); got != want {
		t.Fatalf("r = %d, want %d", got, want)
	}

	if got, want := g, uint32(28270); got != want {
		t.Fatalf("g = %d, want %d", got, want)
	}

	if got, want := b, uint32(23901); got != want {
		t.Fatalf("b = %d, want %d", got, want)
	}

	if got, want := a, uint32(65535); got != want {
		t.Fatalf("a = %d, want %d", got, want)
	}
}

func TestLayerNRGBAAt(t *testing.T) {
	l := newTestLayer()

	if got, want := l.NRGBAAt(-1, 100), ColorTransparent; got != want {
		t.Fatalf("l.NRGBAAt(-1, 100) = %v, want %v", got, want)
	}

	if got, want := l.NRGBAAt(2, 2), ColorNRGBA(66, 110, 93, 255); got != want {
		t.Fatalf("l.NRGBAAt(2, 2) = %v, want %v", got, want)
	}
}

func TestLayerAlphaAt(t *testing.T) {
	l := newTestLayer()

	if got, want := l.AlphaAt(-1, 100), uint8(0); got != want {
		t.Fatalf("l.AlphaAt(-1, 100) = %d, want %d", got, want)
	}

	if got, want := l.AlphaAt(0, 11), uint8(255); got != want {
		t.Fatalf("l.AlphaAt(0,12) = %d, want %d", got, want)
	}
}

func TestLayerBounds(t *testing.T) {
	l := newTestLayer()

	r := l.Bounds()

	if got, want := r.Dx(), 16; got != want {
		t.Fatalf("r.Dx() = %d, want %d", got, want)
	}

	if got, want := r.Dy(), 12; got != want {
		t.Fatalf("r.Dy() = %d, want %d", got, want)
	}
}

func TestLayerColorModel(t *testing.T) {
	l := newTestLayer()

	if l.ColorModel() != color.RGBAModel {
		t.Fatalf("unexpected color model")
	}
}

func TestLayerColorIndexAt(t *testing.T) {
	l := newTestLayer()

	if got, want := l.ColorIndexAt(1, 1), uint8(1); got != want {
		t.Fatalf("l.ColorIndexAt(3,3) = %d, want %d", got, want)
	}

	if got, want := l.ColorIndexAt(6, 6), uint8(3); got != want {
		t.Fatalf("l.ColorIndexAt(6,6) = %d, want %d", got, want)
	}
}

func TestLayerTilesize(t *testing.T) {
	l := newTestLayer()

	if got, want := l.TileSize(), Pt(4, 4); got != want {
		t.Fatalf("l.TileSize() = %v, want %v", got, want)
	}
}

func TestLayerGfxPalette(t *testing.T) {
	l := newTestLayer()

	if !reflect.DeepEqual(l.GfxPalette(), PaletteEN4) {
		t.Fatalf("l.GfxPalette() returned unexpected palette")
	}
}

func TestLayerColorPalette(t *testing.T) {
	l := newTestLayer()

	if got, want := len(l.ColorPalette()), 4; got != want {
		t.Fatalf("len(l.ColorPalette()) = %d, want %d", got, want)
	}
}

func TestLayerDataAt(t *testing.T) {
	l := newTestLayer()

	if got, want := l.DataAt(1, 1), 1; got != want {
		t.Fatalf("l.DataAt(1,1) = %d, want %d", got, want)
	}

	if got, want := l.DataAt(2, 0), 0; got != want {
		t.Fatalf("l.DataAt(2,0) = %d, want %d", got, want)
	}
}

func TestLayerPut(t *testing.T) {
	l := newTestLayer()

	if got, want := l.Index(0, 0), 0; got != want {
		t.Fatalf("l.Index(1,1) = %d, want %d", got, want)
	}

	l.Put(0, 0, 1)

	if got, want := l.Index(0, 0), 1; got != want {
		t.Fatalf("l.Index(1,1) = %d, want %d", got, want)
	}
}

func TestLayerSetTileIndex(t *testing.T) {
	l := newTestLayer()

	if got, want := l.TileIndexAt(0, 0), 0; got != want {
		t.Fatalf("l.TileIndexAt(1,1) = %d, want %d", got, want)
	}

	l.SetTileIndex(0, 0, 1)

	if got, want := l.TileIndexAt(0, 0), 1; got != want {
		t.Fatalf("l.TileIndexAt(1,1) = %d, want %d", got, want)
	}
}

func TestDataOffset(t *testing.T) {
	for _, tc := range []struct {
		width int
		size  image.Point
		input image.Point
		want  int
	}{
		{10, Pt(10, 10), Pt(20, 5), 70},
		{30, Pt(30, 5), Pt(20, 10), 320},
	} {
		l := &Layer{Width: tc.width, Tileset: &Tileset{Size: tc.size}}

		if got := l.dataOffset(tc.input.X, tc.input.Y); got != tc.want {
			t.Fatalf("l.indexAt(%d, %d) = %d, want %d",
				tc.input.X, tc.input.Y, got, tc.want)
		}
	}
}

func newTestLayer() *Layer {
	ts := NewTileset(PaletteEN4, Pt(4, 4), TilesetData{
		{
			0, 0, 0, 0,
			1, 1, 1, 1,
			2, 2, 2, 2,
			3, 3, 3, 3,
		},
		{
			0, 2, 1, 0,
			0, 3, 3, 0,
			0, 3, 3, 0,
			0, 1, 2, 0,
		},
	})

	return NewLayer(ts, 4, LayerData{
		0, 0, 0, 0,
		0, 1, 1, 0,
		0, 1, 1, 0,
	})
}
