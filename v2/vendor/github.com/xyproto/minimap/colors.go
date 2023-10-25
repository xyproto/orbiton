package minimap

import (
	"errors"
	"strings"

	"github.com/xyproto/mode"
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

// ColorMinimap returns a colored text representation of the given text.
// width and height are the number of characters and lines for the minimap.
// highlightIndex is the line index to be highlighted in the minimap. Use -1 for no highlight.
func ColorMinimap(data string, width, height int, m mode.Mode, highlightIndex int) (string, error) {
	if width <= 0 || height <= 0 {
		return "", errors.New("width and height must both be positive integers")
	}

	lines := strings.Split(data, "\n")

	widthStep := max(1, len(lines[0])/width)
	heightStep := max(1, len(lines)/height)

	var result strings.Builder
	o := textoutput.New()

	determineColor := func(char string, m mode.Mode) string {
		if char == " " {
			return "black" // spaces between tokens in the minimap
		}
		return "darkgray" // default color
	}

	for i := 0; i < min(len(lines), height*heightStep); i += heightStep {
		for j := 0; j < min(len(lines[i]), width*widthStep); j += widthStep {
			color := determineColor(string(lines[i][j]), m)

			// Highlight using LightYellow if the line is within the highlight range
			if i <= highlightIndex && highlightIndex < i+heightStep {
				result.WriteString(o.LightYellow("█"))
			} else {
				result.WriteString(o.Tags("<" + color + ">█<off>"))
			}
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}

// DrawMinimap draws a colored representation of the given text onto a vt100.Canvas.
// x and y are the starting coordinates on the canvas.
// width and height are the number of characters and lines for the minimap.
// highlightIndex is the line index to be highlighted in the minimap. Use -1 for no highlight.
func DrawMinimap(c *vt100.Canvas, data string, x, y, width, height int, m mode.Mode, highlightIndex int, contentColor, spaceColor, highlightColor, bgColor vt100.AttributeColor) error {
	if width <= 0 || height <= 0 {
		return errors.New("width and height must both be positive integers")
	}

	lines := strings.Split(data, "\n")

	widthStep := max(1, len(lines[0])/width)
	heightStep := max(1, len(lines)/height)

	for i := 0; i < min(len(lines), height*heightStep); i += heightStep {
		for j := 0; j < min(len(lines[i]), width*widthStep); j += widthStep {
			char := string(lines[i][j])
			color := contentColor
			if char == " " {
				color = spaceColor
			}
			if i <= highlightIndex && highlightIndex < i+heightStep {
				c.WriteRune(uint(x+j/widthStep), uint(y+i/heightStep), highlightColor, bgColor, '█')
			} else {
				c.WriteRune(uint(x+j/widthStep), uint(y+i/heightStep), color, bgColor, '█')
			}
		}
	}

	return nil
}
