package palgen

import (
	"bytes"
	"image/color"
	"io/ioutil"
)

// ACT converts a given palette to the Photoshop ACT Palette Format (.act)
// There is no header, just either 768 or 772 bytes of color data.
// 256 * 3 = 768. The four extra bytes can be 16-bit color count + 16 bit transparent color index.
func ACT(pal color.Palette) []byte {
	var buf bytes.Buffer
	// Output the colors
	for _, c := range pal {
		cn := c.(color.RGBA)
		buf.WriteByte(cn.R)
		buf.WriteByte(cn.G)
		buf.WriteByte(cn.B)
	}
	// Return the generated string
	return buf.Bytes()
}

// SaveACT can save a palette to file in the Photoship ACT Palette Format (.act)
func SaveACT(pal color.Palette, filename string) error {
	return ioutil.WriteFile(filename, ACT(pal), 0644)
}
