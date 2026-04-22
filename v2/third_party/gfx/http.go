//go:build !tinygo
// +build !tinygo

package gfx

import (
	"errors"
	"image"
)

// Local Orbiton fork: the upstream file imports net/http to provide
// convenience functions (Get, GetPNG, GetImage, GetTileset) that Orbiton
// does not use. Replacing them with error-returning stubs avoids pulling
// net/http, crypto/tls and their dependencies into the binary while still
// satisfying internal references (e.g. from geo.go).

var errNetDisabled = errors.New("gfx: network functions disabled in this build")

// Get is a no-op stub returning an error.
func Get(rawurl string) ([]byte, error) {
	return nil, errNetDisabled
}

// GetPNG is a no-op stub returning an error.
func GetPNG(rawurl string) (image.Image, error) {
	return nil, errNetDisabled
}

// GetImage is a no-op stub returning an error.
func GetImage(rawurl string) (image.Image, error) {
	return nil, errNetDisabled
}

// GetTileset is a no-op stub returning an error.
func GetTileset(size Vec, rawurl string) (*Tileset, error) {
	return nil, errNetDisabled
}
