package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"

	"github.com/xyproto/burnfont"
)

// Render will render the current text to a .png image
func (e *Editor) Render(filename string) error {
	maxlen := 0
	for i := 0; i < e.Len(); i++ {
		line := e.Line(i)
		if len(line) > maxlen {
			maxlen = len(line)
		}
	}

	lineHeight := 14

	marginRight := 4 * lineHeight
	width := maxlen*8 + marginRight
	height := (e.Len()+1)*lineHeight + lineHeight
	textImage := image.NewRGBA(image.Rectangle{image.ZP, image.Point{width, height}})

	cyan := color.NRGBA{0x25, 0x96, 0xd1, 0xff}
	black := color.NRGBA{0, 0, 0, 0xff}

	finalImage := image.NewRGBA(image.Rectangle{image.ZP, image.Point{width, height}})
	draw.Draw(finalImage, finalImage.Bounds(), &image.Uniform{black}, image.ZP, draw.Src)

	var contents string
	for i := 0; i < e.Len(); i++ {
		contents = strings.Replace(e.Line(i), "\t", strings.Repeat(" ", e.spacesPerTab), -1)
		burnfont.DrawString(textImage, lineHeight, (i+1)*lineHeight, contents, cyan)
	}

	// Now overlay the text image on top of the final image with the background color

	draw.Draw(finalImage, finalImage.Bounds(), textImage, image.ZP, draw.Over)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	return png.Encode(f, finalImage)
}
