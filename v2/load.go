package main

import (
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	ico "github.com/dkua/go-ico"
	"github.com/gen2brain/jpegxl"
	bmp "github.com/jsummers/gobmp"
	"github.com/xfmoulet/qoi"
	"golang.org/x/image/webp"
)

// LoadImage loads an image and converts it to *image.NRGBA.
// Currently, PNG, JPEG, ICO, GIF, BMP, and WebP images are supported.
func LoadImage(filename string) (*image.NRGBA, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var img image.Image
	// Read and decode the image
	switch filepath.Ext(strings.ToLower(filename)) {
	case ".bmp":
		img, err = bmp.Decode(f)
	case ".gif":
		img, err = gif.Decode(f)
	case ".ico":
		img, err = ico.Decode(f)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
	case ".jxl":
		img, err = jpegxl.Decode(f)
	case ".png":
		img, err = png.Decode(f)
	case ".qoi":
		img, err = qoi.Decode(f)
	case ".webp":
		img, err = webp.Decode(f)
	}
	if err != nil {
		return nil, err
	}
	if nImage, ok := img.(*image.NRGBA); ok {
		return nImage, nil
	}
	return ConvertToNRGBA(img)
}
