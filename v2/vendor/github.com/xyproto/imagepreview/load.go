package imagepreview

import (
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	ico "github.com/dkua/go-ico"
	bmp "github.com/jsummers/gobmp"
	"github.com/xfmoulet/qoi"
	"golang.org/x/image/webp"
)

// LoadImage loads an image and converts it to *image.NRGBA.
// PNG, JPEG, ICO, GIF, BMP, WebP and QOI images are supported.
func LoadImage(filename string) (*image.NRGBA, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var img image.Image
	switch filepath.Ext(strings.ToLower(filename)) {
	case ".bmp":
		img, err = bmp.Decode(f)
	case ".gif":
		img, err = gif.Decode(f)
	case ".ico":
		img, err = ico.Decode(f)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
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

// ConvertToNRGBA converts the given image.Image to *image.NRGBA.
func ConvertToNRGBA(img image.Image) (*image.NRGBA, error) {
	nImage := image.NewNRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c, ok := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if !ok {
				return nil, errors.New("could not convert color to NRGBA")
			}
			nImage.Set(x, y, c)
		}
	}
	return nImage, nil
}
