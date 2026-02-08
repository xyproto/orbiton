package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/xyproto/palgen"
	"github.com/xyproto/vt"
)

var drawRune = func() rune {
	if useASCII {
		return '#'
	}
	return '▒'
}()

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
func Draw(canvas *vt.Canvas, m image.Image) error {
	// Convert the image to only use the basic 16-color palette
	img, err := palgen.ConvertBasic(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	// img is now an indexed image
	var vc vt.AttributeColor
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			for i, rgb := range palgen.BasicPalette16 {
				if rgb[0] == c.R && rgb[1] == c.G && rgb[2] == c.B {
					switch i {
					case 0:
						vc = vt.Black
					case 1:
						vc = vt.Red
					case 2:
						vc = vt.Green
					case 3:
						vc = vt.Yellow
					case 4:
						vc = vt.Blue
					case 5:
						vc = vt.Magenta
					case 6:
						vc = vt.Cyan
					case 7:
						vc = vt.LightGray
					case 8:
						vc = vt.DarkGray
					case 9:
						vc = vt.LightRed
					case 10:
						vc = vt.LightGreen
					case 11:
						vc = vt.LightYellow
					case 12:
						vc = vt.LightBlue
					case 13:
						vc = vt.LightMagenta
					case 14:
						vc = vt.LightCyan
					case 15:
						vc = vt.White
					default:
						vc = vt.White
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
