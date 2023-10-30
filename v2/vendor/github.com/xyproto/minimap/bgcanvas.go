package minimap

import (
	"errors"
	"strings"

	"github.com/xyproto/vt100"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
	widthStep := max(1, len(lines[0])/width)
	heightStep := max(1, len(lines)/height)

	if highlightIndex < 0 || highlightIndex >= len(lines) {
		// Set highlight to the middle if out of bounds or set to -1
		highlightIndex = len(lines) / 2
	}
	representativeHighlight := highlightIndex / heightStep

	for i := 0; i < min(len(lines), height*heightStep); i += heightStep {
		minimapLine := i / heightStep
		currentLine := normalizeLine(lines[i], width*widthStep)
		for j := 0; j < min(len(currentLine), width*widthStep); j += widthStep {
			char := string(currentLine[j])
			color := contentColor
			if char == " " {
				color = spaceColor
			}
			if minimapLine == representativeHighlight {
				c.WriteBackground(uint(x+j/widthStep), uint(y+i/heightStep), highlightColor)
			} else {
				c.WriteBackground(uint(x+j/widthStep), uint(y+i/heightStep), color)
			}
		}
	}
	return nil
}
