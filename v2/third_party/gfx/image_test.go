package gfx

import (
	"image"
	"testing"
)

func TestPt(t *testing.T) {
	got, want := Pt(1, 2), image.Point{1, 2}

	if got != want {
		t.Fatalf("Pt(1,2) = %v, want %v", got, want)
	}
}

func TestIR(t *testing.T) {
	x0, y0, x1, y1 := 10, 10, 30, 30

	got := IR(x0, y0, x1, y1)
	want := IR(x0, y0, x1, y1)

	if got != want {
		t.Fatalf("IR(%d, %d, %d, %d) = %v, want %v", x0, y0, x1, y1, got, want)
	}
}
