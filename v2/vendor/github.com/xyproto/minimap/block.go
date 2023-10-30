package minimap

import (
	"strings"
)

// Block creates a minimap of the given text, using block characters
func Block(contents string, targetLineLength, targetOutputLines int) string {
	// If targetOutputLines is 0, return an empty string
	if targetOutputLines == 0 {
		return ""
	}

	intermediateMap := Simple(contents, targetLineLength, 2*targetOutputLines)

	// Convert the intermediate map to the dual representation
	lines := strings.Split(intermediateMap, "\n")

	var sb strings.Builder
	for i := 0; i < len(lines); i += 2 {
		for j := 0; j < targetLineLength; j++ {
			upper := false
			lower := false
			if j < len(lines[i]) && lines[i][j] == '*' {
				upper = true
			}
			if i+1 < len(lines) && j < len(lines[i+1]) && lines[i+1][j] == '*' {
				lower = true
			}

			switch {
			case upper && !lower:
				sb.WriteRune('▀') // upper half block
			case !upper && lower:
				sb.WriteRune('▄') // lower half block
			case upper && lower:
				sb.WriteRune('█') // full block
			default:
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}
	return strings.TrimSpace(sb.String())
}
