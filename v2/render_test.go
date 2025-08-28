package main

import (
	"testing"
)

func TestRender(_ *testing.T) {
	// Just check that .ToImage() is available and possible to call
	NewCanvas().ToImage()
}
