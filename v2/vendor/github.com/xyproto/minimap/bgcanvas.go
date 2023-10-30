package minimap

import (
	"errors"
	"strings"

	"github.com/xyproto/vt100"
)

func normalizeLine(line string, length int) string {
	if len(line) > length {
		return line[:length]
	}
	return line + strings.Repeat(" ", length-len(line))
}

// DrawBackgroundMinimap draws a colored representation of the given text onto a vt100.Canvas.
func DrawBackgroundMinimap(c *vt100.Canvas, data string, x, y, width, height int, highlightIndex int, contentColor, spaceColor, highlightColor vt100.AttributeColor) error {
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

		// Determine the representative line for this minimap line
		srcLineIndex := startSrcLineIndex
		currentLine := normalizeLine(lines[srcLineIndex], width)

		for j := 0; j < width; j++ {
			srcCharIndex := j * len(currentLine) / width
			char := string(currentLine[srcCharIndex])
			color := contentColor
			if char == " " {
				color = spaceColor
			}
			if highlightIndex >= startSrcLineIndex && highlightIndex < endSrcLineIndex {
				c.WriteBackground(uint(x+j), uint(y+i), highlightColor)
			} else {
				c.WriteBackground(uint(x+j), uint(y+i), color)
			}
		}
	}
	return nil
}
