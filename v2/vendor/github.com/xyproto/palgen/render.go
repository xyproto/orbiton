package palgen

import (
	"image"
	"image/color"
)

// Render a given palette as an image, with blocks of 16x16 pixels and 32 colors per row of blocks.
func Render(pal color.Palette) image.Image {
	return RenderWithConfig(pal, 16, 32)
}

// RenderWithConfig can render a given palette as an image that shows each color as a square, lined up in rows.
// blockSize is the size of the color block per color, in pixels (16 is default)
// colorsPerRow is the number of color blocks per row (32 is default)
func RenderWithConfig(pal color.Palette, blockSize, colorsPerRow int) image.Image {

	// Remove the alpha
	for i, c := range pal {
		rgba := color.RGBAModel.Convert(c).(color.RGBA)
		rgba.A = 255
		pal[i] = rgba
	}

	// The first color is now the darkest one
	darkIndex := uint8(0)

	// Let each row have colorsPerRow blocks
	w := colorsPerRow
	h := len(pal) / w
	leftover := len(pal) % w
	if leftover > 0 {
		h++
	}

	// "pixel" width and height
	pw := blockSize
	ph := blockSize

	// TODO: Clean up everything that has to do with borderSize and support
	//       custom border sizes.

	// size of border, ish
	borderSize := 2

	upLeft := image.Point{0, 0}
	lowRight := image.Point{w*pw + (borderSize-1)*2, h*ph + (borderSize-1)*2}

	// Create a new image, where a square is painted per color in the palette
	palImage := image.NewPaletted(image.Rectangle{upLeft, lowRight}, pal)

	for y := upLeft.Y; y < lowRight.Y; y++ {
		for x := lowRight.X; x < lowRight.X; x++ {
			if x < (borderSize-1) || y < (borderSize-1) || x >= (lowRight.X-(borderSize-1)) || y >= (lowRight.Y-(borderSize-1)) {
				palImage.SetColorIndex(x, y, darkIndex)
			}
		}
	}

	// Set color for each pixel.
	colorIndex := uint8(0)
OUT:
	// For each palette color block
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// For each pixel in the block of size pw x ph
			for by := 0; by < ph; by++ {
				for bx := 0; bx < pw; bx++ {
					if bx <= borderSize-2 || by <= borderSize-2 || bx >= pw-(borderSize-1) || by >= ph-(borderSize-1) {
						palImage.SetColorIndex((x*pw)+bx+(borderSize-1), (y*ph)+by+(borderSize-1), darkIndex)
					} else {
						palImage.SetColorIndex((x*pw)+bx+(borderSize-1), (y*ph)+by+(borderSize-1), colorIndex)
					}
				}
			}
			colorIndex++
			if int(colorIndex) >= len(pal) {
				break OUT
			}
		}
	}

	return palImage
}
