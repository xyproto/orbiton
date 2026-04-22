//go:build !tinygo
// +build !tinygo

package gfx

import (
	"bytes"
	"encoding/base64"
	"image"
)

// Base64EncodedPNG encodes the given image into
// a string using base64.StdEncoding.
func Base64EncodedPNG(src image.Image) string {
	var buf bytes.Buffer

	EncodePNG(&buf, src)

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// Base64ImgTag returns a HTML tag for an img
// with its src set to a base64 encoded PNG.
func Base64ImgTag(src image.Image) string {
	return `<img src="data:image/png;base64,` + Base64EncodedPNG(src) + `">`
}
