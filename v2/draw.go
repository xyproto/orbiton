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

// paletteColorMap maps RGB values to VT100 attribute colors for fast lookup
var paletteColorMap map[[3]uint8]vt.AttributeColor

func init() {
	vtColors := []vt.AttributeColor{
		vt.Black, vt.Red, vt.Green, vt.Yellow,
		vt.Blue, vt.Magenta, vt.Cyan, vt.LightGray,
		vt.DarkGray, vt.LightRed, vt.LightGreen, vt.LightYellow,
		vt.LightBlue, vt.LightMagenta, vt.LightCyan, vt.White,
	}
	paletteColorMap = make(map[[3]uint8]vt.AttributeColor, len(palgen.BasicPalette16))
	for i, rgb := range palgen.BasicPalette16 {
		if i < len(vtColors) {
			paletteColorMap[rgb] = vtColors[i]
		}
	}
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
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			vc := vt.White // default
			if found, ok := paletteColorMap[[3]uint8{c.R, c.G, c.B}]; ok {
				vc = found
			}
			// Draw the "pixel" on the canvas using the vc color and the draw rune
			canvas.PlotColor(uint(x), uint(y), vc, drawRune)
		}
	}
	return nil
}
