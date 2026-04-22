package gfx

import (
	"image/color"
	"testing"
)

func TestColorWithAlpha(t *testing.T) {
	c := ColorWithAlpha(ColorRGBA(255, 0, 0, 255), 128)

	if got, want := c.A, uint8(128); got != want {
		t.Fatalf("c.A = %d, want %d", got, want)
	}
}

func TestColorNRGBA(t *testing.T) {
	got := ColorNRGBA(11, 22, 33, 44)
	want := color.NRGBA{11, 22, 33, 44}

	if got != want {
		t.Fatalf("ColorNRGBA(11,22,33,44) = %v, want %v", got, want)
	}
}

func TestColorRGBA(t *testing.T) {
	got := ColorRGBA(11, 22, 33, 44)
	want := color.RGBA{11, 22, 33, 44}

	if got != want {
		t.Fatalf("ColorRGBA(11,22,33,44) = %v, want %v", got, want)
	}
}

func TestLerpColors(t *testing.T) {
	for _, tc := range []struct {
		t float64
		r uint32
	}{
		{-10, 65535},
		{0.0, 65535},
		{0.1, 58981},
		{0.5, 32767},
		{0.9, 6553},
		{1.0, 0},
		{100, 0},
	} {
		c := LerpColors(ColorWhite, ColorBlack, tc.t)

		if r, _, _, _ := c.RGBA(); r != tc.r {
			t.Fatalf("c.R = %d, want %d", r, tc.r)
		}
	}
}
