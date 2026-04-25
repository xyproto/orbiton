//go:build !tinygo
// +build !tinygo

package gfx

import "image"

// Playground displays image on The Go Playground
// using the IMAGE: base64 encoded PNG “hack”
func Playground(src image.Image) {
	Log("IMAGE: %s", Base64EncodedPNG(src))
}
