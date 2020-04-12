package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"strings"

	ico "github.com/biessek/golang-ico"
)

var (
	// 4-bit, 16-color grayscale grading by runes
	lookupRunes = map[rune]byte{
		' ': 0,
		'.': 1,
		',': 2,
		'-': 3,
		'~': 4,
		':': 5,
		'=': 6,
		';': 7,
		'+': 8,
		'x': 9,
		'*': 10,
		'?': 11,
		'%': 12,
		'@': 13,
		'#': 14,
		'W': 15,
	}
)

// ReadFavicon will try to load an .ico file into a "\n" separated []byte
// Returns a Mode (representing: 16 color grayscale, rgb or rgba), the textual representation and an error
// If dummy is true, the textual representation of a blank 16 color grayscale image will be returned.
// May return a warning/message string as well.
func ReadFavicon(filename string, dummy bool) (Mode, []byte, string, error) {
	var (
		mode    Mode = modeBlank
		m       image.Image
		bounds  image.Rectangle
		buf     bytes.Buffer
		message string
	)

	if dummy {
		// Create a dummy image, 16x16, all gray
		tm := image.NewNRGBA(image.Rect(0, 0, 16, 16))
		bounds = tm.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				tm.Set(x, y, color.NRGBA{127, 127, 127, 255})
			}
		}
		m = tm
	} else {
		// Read the file
		reader, err := os.Open(filename)
		if err != nil {
			return mode, []byte{}, "", err
		}
		defer reader.Close()

		// Decode the image
		icoImage, err := ico.Decode(reader)
		if err != nil {
			return mode, []byte{}, "", err
		}

		m = icoImage
	}

	// Check the size of the image
	// TODO: Consider lifting this restriction
	if m.Bounds() != image.Rect(0, 0, 16, 16) {
		return mode, []byte{}, "", errors.New("Only 16x16 images are supported")
	}

	lookupLetters := make(map[byte]rune)
	for key, value := range lookupRunes {
		lookupLetters[value] = key
	}

	if m.ColorModel() != color.GrayModel {
		// Warning message
		message = "will convert to 4-bit grayscale when saving"
	}

	// Convert the image to a textual representation
	bounds = m.Bounds()
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
					buf.WriteRune(lookupLetters[byte(luma16)])
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
		buf.WriteString("\n")
		for i := byte(0); i < byte(16); i++ {
			buf.WriteString(fmt.Sprintf("%2d = %c\n", i, lookupLetters[i]))
		}
	}
	return mode, buf.Bytes(), message, nil
}

// WriteFavicon converts the textual representation to an .ico image
func WriteFavicon(mode Mode, text, filename string) error {
	if mode != modeGray4 {
		return errors.New("Saving .ico files is only implenented for 4-bit grayscale images")
	}

	var (
		// Create a new image
		width  = 16
		height = 16
		m      = image.NewRGBA(image.Rect(0, 0, width, height))

		// These are used in the loops below
		x, y      int
		line      string
		intensity byte
		r         rune
		runes     []rune
	)

	// Draw the pixels
	for y, line = range strings.Split(text, "\n") {
		if y >= 16 { // max 16x16 pixels
			break
		}
		runes = []rune(line)
		for x = 0; x < 16; x++ { // max 16x16 pixels
			if (x * 2) < len(runes) {
				r = runes[x*2]
				intensity = lookupRunes[r]*16 + 15 // from 0..15 to 15..255
				// Draw pixel to image
				m.Set(x, y, color.RGBA{intensity, intensity, intensity, 0xff})
			} else {
				// Draw a transparent pixel
				m.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}

	// Create a new file
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	// Encode the image as an .ico image
	//return ico.Encode(f, m)
	return EncodeGrayscale4bit(f, m)
}

// This is from github.com/biessek/golang-ico, only to be able to use private structs
type head struct {
	Zero   uint16
	Type   uint16
	Number uint16
}

// This is from github.com/biessek/golang-ico, only to be able to use private structs
type direntry struct {
	Width   byte
	Height  byte
	Palette byte
	_       byte
	Plane   uint16
	Bits    uint16
	Size    uint32
	Offset  uint32
}

// EncodeGrayscale4bit is a modified version of the function from github.com/biessek/golang-ico, only to be able to save 4-bit .ico images
func EncodeGrayscale4bit(w io.Writer, im image.Image) error {
	b := im.Bounds()
	m := image.NewGray(b)
	draw.Draw(m, b, im, b.Min, draw.Src)
	header := head{
		0,
		1,
		1,
	}
	entry := direntry{
		Plane:  1,
		Bits:   4, // was: 32
		Offset: 22,
	}
	pngbuffer := new(bytes.Buffer)
	pngwriter := bufio.NewWriter(pngbuffer)
	err := png.Encode(pngwriter, m)
	if err != nil {
		return err
	}
	err = pngwriter.Flush()
	if err != nil {
		return err
	}
	entry.Size = uint32(len(pngbuffer.Bytes()))
	bounds := m.Bounds()
	entry.Width = uint8(bounds.Dx())
	entry.Height = uint8(bounds.Dy())
	bb := new(bytes.Buffer)
	var e error
	if e = binary.Write(bb, binary.LittleEndian, header); e != nil {
		return e
	}
	if e = binary.Write(bb, binary.LittleEndian, entry); e != nil {
		return e
	}
	if _, e = w.Write(bb.Bytes()); e != nil {
		return e
	}
	_, e = w.Write(pngbuffer.Bytes())
	return e
}
