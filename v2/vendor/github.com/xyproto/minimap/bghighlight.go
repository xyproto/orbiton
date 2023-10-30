package minimap

import (
	"errors"
	"strings"

	"github.com/xyproto/vt100"
)

// DrawBackgroundMinimapHighlight draws just the highlighted line of the minimap
func DrawBackgroundMinimapHighlight(c *vt100.Canvas, data string, x, y, width, height int, highlightIndex int, highlightColor vt100.AttributeColor) error {
	if width <= 0 || height <= 0 {
		return errors.New("width and height must both be positive integers")
	}
	lines := strings.Split(data, "\n")

	if highlightIndex < 0 || highlightIndex >= len(lines) {
		// Set highlight to the middle if out of bounds or set to -1
		highlightIndex = len(lines) / 2
	}

	for i := 0; i < height; i++ {
		startSrcLineIndex := i * len(lines) / height
		endSrcLineIndex := (i + 1) * len(lines) / height

		for j := 0; j < width; j++ {
			if highlightIndex >= startSrcLineIndex && highlightIndex < endSrcLineIndex {
				c.WriteBackground(uint(x+j), uint(y+i), highlightColor)
			}
		}
	}
	return nil
}
