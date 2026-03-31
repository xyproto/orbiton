package imagepreview

import (
	"image"
	"image/color"
	"testing"

	"github.com/xyproto/vt"
)

func TestRenderWithoutColor(t *testing.T) {
	// Create a simple test image (4x4)
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))

	// Fill with some test colors
	img.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})   // Red
	img.SetRGBA(1, 0, color.RGBA{0, 255, 0, 255})   // Green
	img.SetRGBA(2, 0, color.RGBA{0, 0, 255, 255})   // Blue
	img.SetRGBA(3, 0, color.RGBA{255, 255, 0, 255}) // Yellow

	// Save original IsVT value
	original := IsVT
	defer func() { IsVT = original }()

	// Test with colors enabled
	IsVT = false
	canvas1 := vt.NewCanvas()
	err := DrawOnCanvas(canvas1, img, BlockRune)
	if err != nil {
		t.Fatalf("DrawOnCanvas failed with colors: %v", err)
	}
	t.Logf("Canvas with colors rendered successfully")

	// Test without colors (vt100)
	IsVT = true
	canvas2 := vt.NewCanvas()
	err = DrawOnCanvas(canvas2, img, BlockRune)
	if err != nil {
		t.Fatalf("DrawOnCanvas failed without colors: %v", err)
	}
	t.Logf("Canvas without colors rendered successfully")
}
