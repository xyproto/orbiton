package main

import (
	"testing"
)

func TestToImage(_ *testing.T) {
	// Just check that .ToImage() is available and possible to call
	NewCanvas().ToImage()
}
