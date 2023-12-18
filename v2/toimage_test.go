package main

import (
	"testing"

	"github.com/xyproto/vt100"
)

func TestToImage(_ *testing.T) {
	// Just check that .ToImage() is available and possible to call
	vt100.NewCanvas().ToImage()
}
