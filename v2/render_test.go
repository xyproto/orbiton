package main

import (
	"testing"

	"github.com/xyproto/vt"
)

func TestRender(_ *testing.T) {
	// Just check that .ToImage() is available and possible to call
	vt.NewCanvas().ToImage()
}
