package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"strings"

	"github.com/biessek/golang-ico"
)

// ReadFavicon will try to load an .ico file into a "\n" separated []byte
// Returns a Mode (representing: 16 color grayscale, rgb or rgba), the textual representation and an error
func ReadFavicon(filename string) (Mode, []byte, error) {

	var mode Mode = modeBlank

	// Read the file
	reader, err := os.Open(filename)
	if err != nil {
		return mode, []byte{}, err
	}
	defer reader.Close()

	// Decode the image
	icoImage, err := ico.Decode(reader)
	if err != nil {
		return mode, []byte{}, err
	}

	// Convert the image to NRGBA
	m, ok := icoImage.(*image.NRGBA)
	if !ok {
		return mode, []byte{}, errors.New("not NRGBA")
	}

	// Check the size of the image
	// TODO: Consider lifting this restriction
	if m.Bounds() != image.Rect(0, 0, 16, 16) {
		return mode, []byte{}, errors.New("Only 16x16 .ico files are supported")
	}

	const (
		// 4-bit, 16-color grayscale grading by runes
		letters = " .,-~:=;+x*?%@#W"
	)

	// Convert the image to a textual representation
	var buf bytes.Buffer
	bounds := m.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := m.At(x, y).RGBA()
			// Found a luma formula here: https://riptutorial.com/go/example/31693/convert-color-image-to-grayscale
			luma := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) * (255.0 / 65535)

			// luma16 is 0..15
			luma16 := int(math.Round(luma) / 16.0)
			if luma16 > 15 {
				luma16 = 15
			}

			mode = modeGray4 // 4-bit grayscale, 16 different color values

			if mode == modeGray4 {
				// 4-bit grayscale
				if luma16 == 0 {
					buf.WriteString("  ")
				} else {
					buf.WriteRune([]rune(letters)[luma16])
					buf.Write([]byte{' '}) // Add a space, to make the proportions look better
				}
			} else if mode == modeRGB {
				// 8+8+8 bit RGB
				if r+g+b+a == 0 {
					buf.WriteString("|      ")
				} else {
					buf.WriteString(strings.Replace(fmt.Sprintf("|%2x%2x%2x", r/256, g/256, b/256), " ", "0", -1))
				}
			} else if mode == modeRGBA {
				// 8+8+8+8 bit RGBA
				if r+g+b+a == 0 {
					buf.WriteString("|        ")
				} else {
					buf.WriteString(strings.Replace(fmt.Sprintf("|%2x%2x%2x%2x", r/256, g/256, b/256, a/256), " ", "0", -1))
				}
			}
		}
		if mode != modeGray4 {
			buf.Write([]byte{'|', '\n'})
		}
		// The blank lines are for the proportions to look right
		buf.WriteString("\n")
		if mode == modeRGB {
			buf.WriteString("\n\n")
		}
		if mode == modeRGBA {
			buf.WriteString("\n")
		}
	}
	if mode == modeGray4 {
		// Legend
		for i, r := range letters {
			buf.WriteString(fmt.Sprintf("%2d = %c\n", i, r))
		}
	}
	return mode, buf.Bytes(), nil
}
