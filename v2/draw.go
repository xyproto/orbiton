package main

import (
	"image"

	"github.com/xyproto/imagepreview"
	"github.com/xyproto/vt"
)

var drawRune = func() rune {
	if useASCII {
		return imagepreview.ASCIIRune
	}
	return imagepreview.BlockRune
}()

// Draw attempts to draw the given image.Image onto a VT100 Canvas
func Draw(canvas *vt.Canvas, m image.Image) error {
	return imagepreview.DrawOnCanvas(canvas, m, drawRune)
}
