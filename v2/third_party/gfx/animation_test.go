package gfx

import (
	"bytes"
	"image"
	"testing"
)

func TestAnimationAddPalettedImage(t *testing.T) {
	a := &Animation{}

	a.AddPalettedImage(NewPaletted(3, 3, PaletteEN4))

	if got, want := len(a.Frames), 1; got != want {
		t.Fatalf("len(a.Frames) = %d, want %d", got, want)
	}
}

func TestAnimationEncodeGIF(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		a := &Animation{}

		a.AddPalettedImage(NewPaletted(3, 3, PaletteEN4))

		w := bytes.NewBuffer(nil)

		a.EncodeGIF(w)
	})

	t.Run("Error", func(t *testing.T) {
		a := &Animation{Frames: []image.Image{NewImage(32, 32)}}

		if a.EncodeGIF(nil) == nil {
			t.Fatalf("expected error")
		}
	})
}
