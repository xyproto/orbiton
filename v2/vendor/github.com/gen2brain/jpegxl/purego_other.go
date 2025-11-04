//go:build (!unix && !darwin && !windows) || nodynamic

package jpegxl

import (
	"fmt"
	"image"
	"io"
)

var (
	dynamic    = false
	dynamicErr = fmt.Errorf("jpegxl: dynamic disabled")
)

func decodeDynamic(r io.Reader, configOnly, decodeAll bool) (*JXL, image.Config, error) {
	return nil, image.Config{}, dynamicErr
}

func encodeDynamic(w io.Writer, m image.Image, quality, effort int) error {
	return dynamicErr
}

func loadLibrary() (uintptr, error) {
	return 0, dynamicErr
}
