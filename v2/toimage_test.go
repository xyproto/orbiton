package main

import (
	"testing"

	"github.com/xyproto/vt100"
)

func TestToImage(t *testing.T) {
	vt100.NewCanvas().ToImage()
}
