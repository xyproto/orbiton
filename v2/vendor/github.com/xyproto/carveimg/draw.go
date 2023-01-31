package carveimg

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/xyproto/palgen"
	"github.com/xyproto/vt100"
)

var DrawRune = '▒'

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
func Draw(canvas *vt100.Canvas, m image.Image) error {
	// Convert the image to only use the basic 16-color palette
	img, err := palgen.ConvertBasic(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	// img is now an indexed image
	var vc vt100.AttributeColor
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			for i, rgb := range palgen.BasicPalette16 {
				if rgb[0] == c.R && rgb[1] == c.G && rgb[2] == c.B {
					switch i {
					case 0:
						vc = vt100.Black
					case 1:
						vc = vt100.Red
					case 2:
						vc = vt100.Green
					case 3:
						vc = vt100.Yellow
					case 4:
						vc = vt100.Blue
					case 5:
						vc = vt100.Magenta
					case 6:
						vc = vt100.Cyan
					case 7:
						vc = vt100.LightGray
					case 8:
						vc = vt100.DarkGray
					case 9:
						vc = vt100.LightRed
					case 10:
						vc = vt100.LightGreen
					case 11:
						vc = vt100.LightYellow
					case 12:
						vc = vt100.LightBlue
					case 13:
						vc = vt100.LightMagenta
					case 14:
						vc = vt100.LightCyan
					case 15:
						vc = vt100.White
					default:
						vc = vt100.White
					}
					break
				}
			}
			// Draw the "pixel" on the canvas using the vc color and the draw rune
			canvas.PlotColor(uint(x), uint(y), vc, DrawRune)
		}
	}
	return nil
}
