//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image/gif"
	"image/jpeg"
	"image/png"
)

// Assign all image decode functions to _.
var (
	_ = gif.Decode
	_ = jpeg.Decode
	_ = png.Decode
)
