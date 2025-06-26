package main

import (
	"errors"
	"fmt"
	"image"
	"path/filepath"

	"github.com/xyproto/vt"
	"golang.org/x/image/draw"
)

// displayImage loads and scales an image and tries to draw it to the terminal canvas
func displayImage(c *vt.Canvas, filename string, waitForKeypress bool) error {
	// Find the width and height of the canvas
	width := int(c.Width())
	height := int(c.Height())

	// Load the given filename
	nImage, err := LoadImage(filename)
	if err != nil {
		vt.Close()
		return fmt.Errorf("could not load %s: %s", filename, err)
	}

	imageHeight := float64(nImage.Bounds().Max.Y - nImage.Bounds().Min.Y)
	if imageHeight == 0 {
		return errors.New("the height of the given image is 0")
	}

	imageWidth := float64(nImage.Bounds().Max.X - nImage.Bounds().Min.X)
	if imageWidth == 0 {
		return errors.New("the width of the given image is 0")
	}

	ratio := (imageHeight / imageWidth) * 4.0 // terminal "pixels" are a bit narrow, so multiply by 4.0
	if ratio == 0 {
		return errors.New("the ratio of the given image is 0")
	}

	// Use a smaller width, if that makes the image more like the original proportions
	proportionalWidth := int(float64(height) * ratio)
	if proportionalWidth < width {
		width = proportionalWidth
	}

	// Set the desired size to the size of the current terminal emulator
	resizedImage := image.NewRGBA(image.Rect(0, 0, width, height))

	// Resize the image
	imageResizeFunction := draw.CatmullRom // other alternatives: draw.NearestNeighbor, draw.ApproxBiLinear and draw.BiLinear
	imageResizeFunction.Scale(resizedImage, resizedImage.Rect, nImage, nImage.Bounds(), draw.Over, nil)

	// Draw the image to the canvas, using only the basic 16 colors
	if err := Draw(c, resizedImage); err != nil {
		vt.Close()
		return fmt.Errorf("could not draw image: %s", err)
	}

	// Output the filename on top of the image
	title := " " + filepath.Base(filename) + " "
	c.Write(uint((width-len(title))/2), uint(height-1), vt.Black, vt.BackgroundGray, title)

	// Draw the contents of the canvas to the screen
	c.HideCursorAndDraw()

	// Show the cursor after the keypress
	defer vt.ShowCursor(true)

	if waitForKeypress {
		// Wait for a keypress
		vt.WaitForKey()
	}

	return nil
}
