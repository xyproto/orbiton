package palgen

import (
	"errors"
	"image"
	"image/color"
	"image/color/palette"
)

// TODO: Create a custom Paletted type, based on image/Paletted, that can take > 256 colors

// ConvertCustom can convert an image from True Color to a <=256 color paletted image, given a custom palette.
func ConvertCustom(m image.Image, pal color.Palette) (image.Image, error) {
	if len(pal) > 256 {
		return nil, errors.New("can convert to a maximum of 256 colors")
	}
	palImg := image.NewPaletted(m.Bounds(), pal)
	// For each pixel, go through each color in the palette and pick out the closest one.
	for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
		for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
			sourceColor := m.At(x, y)
			colorIndex := uint8(pal.Index(sourceColor))
			palImg.SetColorIndex(x, y, colorIndex)
		}
	}
	return palImg, nil
}

// Convert can convert an image from True Color to a 256 color paletted image.
// The palette is automatically extracted from the image.
func Convert(m image.Image) (image.Image, error) {
	customPalette, err := Generate(m, 256)
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
	return ConvertCustom(m, customPalette)
}

// ConvertGeneral can convert an image from True Color to a 256 color paletted image, using a general palette.
func ConvertGeneral(m image.Image) (image.Image, error) {
	return ConvertCustom(m, GeneralPalette())
}

// ConvertPlan9 can convert an image from True Color to a 256 color paletted image, using the Plan9 palette from the Go standard library.
func ConvertPlan9(m image.Image) (image.Image, error) {
	return ConvertCustom(m, palette.Plan9)
}

// ConvertBasic can convert an image from True Color to a 16 color paletted image, using the 16 basic terminal colors
func ConvertBasic(m image.Image) (image.Image, error) {
	return ConvertCustom(m, BasicPalette())
}
