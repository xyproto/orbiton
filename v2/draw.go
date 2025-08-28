package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/xyproto/palgen"
)

const drawRune = '▒'

// ConvertToNRGBA converts the given image.Image to *image.NRGBA
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

// Draw attempts to draw the given image.Image onto a VT100 Canvas
func Draw(canvas *Canvas, m image.Image) error {
	// Convert the image to only use the basic 16-color palette
	img, err := palgen.ConvertBasic(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	// img is now an indexed image
	var vc AttributeColor
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			for i, rgb := range palgen.BasicPalette16 {
				if rgb[0] == c.R && rgb[1] == c.G && rgb[2] == c.B {
					switch i {
					case 0:
						vc = Black
					case 1:
						vc = Red
					case 2:
						vc = Green
					case 3:
						vc = Yellow
					case 4:
						vc = Blue
					case 5:
						vc = Magenta
					case 6:
						vc = Cyan
					case 7:
						vc = LightGray
					case 8:
						vc = DarkGray
					case 9:
						vc = LightRed
					case 10:
						vc = LightGreen
					case 11:
						vc = LightYellow
					case 12:
						vc = LightBlue
					case 13:
						vc = LightMagenta
					case 14:
						vc = LightCyan
					case 15:
						vc = White
					default:
						vc = White
					}
					break
				}
			}
			// Draw the "pixel" on the canvas using the vc color and the draw rune
			canvas.PlotColor(uint(x), uint(y), vc, drawRune)
		}
	}
	return nil
}
