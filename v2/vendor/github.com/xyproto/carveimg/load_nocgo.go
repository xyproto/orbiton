//go:build !cgo

package carveimg

import (
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	ico "github.com/biessek/golang-ico"
	bmp "github.com/jsummers/gobmp"
)

// LoadImage loads an image and converts it to *image.NRGBA.
// Currently, PNG, GIF and JPEG images are supported.
func LoadImage(filename string) (*image.NRGBA, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var img image.Image
	// Read and decode the image
	switch filepath.Ext(strings.ToLower(filename)) {
	case ".png":
		img, err = png.Decode(f)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
	case ".ico":
		img, err = ico.Decode(f)
	case ".gif":
		img, err = gif.Decode(f)
	case ".bmp":
		img, err = bmp.Decode(f)
	case ".webp":
		return nil, errors.New("webp is not supported when the program is compiled with CGO_ENABLED=0")
	}
	if err != nil {
		return nil, err
	}
	if nImage, ok := img.(*image.NRGBA); ok {
		return nImage, nil
	}
	return ConvertToNRGBA(img)
}
