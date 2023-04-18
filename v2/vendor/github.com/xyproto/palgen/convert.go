package palgen

import (
	"errors"
	"image"
	"image/color"
	"image/color/palette"

	"github.com/xyproto/burnpal"
)

// TODO: Create a custom Paletted type, based on image/Paletted, that can take > 256 colors

// ConvertCustom can convert an image from True Color to a <=256 color paletted image, given a custom palette.
func ConvertCustom(img image.Image, pal color.Palette) (image.Image, error) {
	if len(pal) > 256 {
		return nil, errors.New("can reduce to a maximum of 256 colors")
	}
	palImg := image.NewPaletted(img.Bounds(), pal)
	// For each pixel, go through each color in the palette and pick out the closest one.
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			sourceColor := img.At(x, y)
			colorIndex := uint8(pal.Index(sourceColor))
			palImg.SetColorIndex(x, y, colorIndex)
		}
	}
	return palImg, nil
}

// Convert can convert an image from True Color to a 256 color paletted image.
// The palette is automatically extracted from the image.
func Convert(img image.Image) (image.Image, error) {
	customPalette, err := Generate(img, 256)
	if err != nil {
		return nil, err
	}
	// This should never happen
	if len(customPalette) > 256 {
		return nil, errors.New("the generated palette has too many colors")
	}
	// Sort using the HCL colorspace
	Sort(customPalette)
	// Return a new Paletted image
	return ConvertCustom(img, customPalette)
}

// Reduce can convert an image from True Color to a N color paletted image.
// The palette is automatically extracted from the image.
func Reduce(img image.Image, n int) (image.Image, error) {
	customPalette, err := GenerateUpTo(img, n)
	if err != nil {
		return nil, err
	}
	// This should never happen
	if len(customPalette) > n {
		return nil, errors.New("the generated palette has too many colors")
	}
	// Sort using the HCL colorspace
	Sort(customPalette)
	// Return a new Paletted image
	return ConvertCustom(img, customPalette)
}

// ConvertGeneral can convert an image from True Color to a 256 color paletted image, using a general palette.
func ConvertGeneral(img image.Image) (image.Image, error) {
	return ConvertCustom(img, GeneralPalette())
}

// ConvertPlan9 can convert an image from True Color to a 256 color paletted image, using the Plan9 palette from the Go standard library.
func ConvertPlan9(img image.Image) (image.Image, error) {
	return ConvertCustom(img, palette.Plan9)
}

// ConvertBasic can convert an image from True Color to a 16 color paletted image, using the 16 basic terminal colors
func ConvertBasic(img image.Image) (image.Image, error) {
	return ConvertCustom(img, BasicPalette())
}

// ConvertBurn can convert an image from True Color to a 256 color paletted image, using the Burn palette from github.com/xyproto/burnpal.
func ConvertBurn(img image.Image) (image.Image, error) {
	return ConvertCustom(img, burnpal.ColorPalette())
}
