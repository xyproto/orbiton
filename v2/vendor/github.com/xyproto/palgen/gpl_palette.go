package palgen

import (
	"errors"
	"fmt"
	"image/color"
	"io/ioutil"
	"strings"
)

const softwareID = "github.com/xyproto/palgen"

// GPL converts a given palette to the GIMP Palette Format (.gpl)
// The given name will be used as the palette name in the header
func GPL(pal color.Palette, paletteName string) (string, error) {
	var sb strings.Builder
	// Prepare a header
	sb.WriteString("GIMP Palette\nName: ")
	sb.WriteString(paletteName)
	sb.WriteString("\nColumns: 4\n# ")
	sb.WriteString(softwareID)
	sb.WriteString("\n")
	// Output the colors
	for i, c := range pal {
		cn, ok := c.(color.RGBA)
		if !ok {
			return "", errors.New("colors in the given palette must be color.RGBA compatible")
		}
		sb.WriteString(fmt.Sprintf("%3d %3d %3d\t%d\n", cn.R, cn.G, cn.B, i))
	}
	// Return the generated string
	return sb.String(), nil
}

// SaveGPL can save a palette to file in the GIMP Palette Format (.gpl)
// The given name will be used as the palette name in the header
func SaveGPL(pal color.Palette, filename, paletteName string) error {
	GPLString, err := GPL(pal, paletteName)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, []byte(GPLString), 0644)
}
