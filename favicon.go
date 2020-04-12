package main

import (
	"bytes"
	"errors"
	"image"
	"math"
	"os"

	"github.com/biessek/golang-ico"
)

// ReadFavicon will try to load an .ico file into a "\n" separated []byte
func ReadFavicon(filename string) ([]byte, error) {
	reader, err := os.Open(filename)
	if err != nil {
		return []byte{}, err
	}
	defer reader.Close()
	//icoImageConfig, err := ico.DecodeConfig(reader)
	//if err != nil {
	//	return []byte{}, nil
	//}
	icoImage, err := ico.Decode(reader)
	if err != nil {
		return []byte{}, err
	}
	m, ok := icoImage.(*image.NRGBA)
	if !ok {
		return []byte{}, errors.New("not NRGBA")
	}
	//if m.Bounds() != image.Rect(0, 0, 16, 16) {
	//	return []byte{}, errors.New("Only 16x16 .ico files are supported")
	//}

	var buf bytes.Buffer
	bounds := m.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			//r, g, b, a := m.At(x, y).RGBA()
			r, g, b, _ := m.At(x, y).RGBA()
			// Found a luma formula here: https://riptutorial.com/go/example/31693/convert-color-image-to-grayscale
			luma := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) * (255.0 / 65535)
			luma16 := int(math.Round(luma) / 16.0)
			if luma16 > 15 {
				luma16 = 15
			}
			// luma16 is 0..15
			//letters := " .:;i!|o%*$LOKMÆ"
			letters := "0123456789ABCDEF"
			buf.Write([]byte{letters[luma16], ' '}) // Add a space, to make the proportions look better
		}
		buf.Write([]byte{'\n'})
	}
	return buf.Bytes(), nil
}
