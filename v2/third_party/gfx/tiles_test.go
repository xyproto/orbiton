package gfx

import "testing"

func TestNewTileset(t *testing.T) {
	ts := NewTileset(PaletteEN4, Pt(4, 4), TilesetData{
		{
			0, 0, 0, 0,
			1, 1, 1, 1,
			2, 2, 2, 2,
			3, 3, 3, 3,
		},
	})

	if got, want := ts.Size.X, 4; got != want {
		t.Fatalf("ts.Size.X = %d, want %d", got, want)
	}
}
