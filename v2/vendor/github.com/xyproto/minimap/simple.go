package minimap

import (
	"strings"
)

// Simple function creates a basic minimap of the contents.
func Simple(contents string, targetLineLength, targetOutputLines int) string {
	if targetOutputLines == 0 {
		return ""
	}

	lines := strings.Split(contents, "\n")
	lenLines := len(lines)
	if lenLines == 0 {
		return ""
	}

	batchSize := lenLines / targetOutputLines
	remainder := lenLines % targetOutputLines
	if batchSize == 0 {
		return strings.Repeat("\n", targetOutputLines-1) // only empty lines
	}

	lineSums := make([]float64, targetOutputLines)
	maxBatchAverage := 0.0

	for i, line := range lines {
		batchIndex := i / batchSize
		if batchIndex >= targetOutputLines {
			batchIndex = targetOutputLines - 1 // for remainders
		}

		lineSums[batchIndex] += float64(len(line))
	}

	for i, sum := range lineSums {
		divider := batchSize
		if i == targetOutputLines-1 && remainder != 0 {
			divider = remainder
		}
		average := sum / float64(divider)
		lineSums[i] = average

		if average > maxBatchAverage {
			maxBatchAverage = average
		}
	}

	scaleDown := float64(targetLineLength) / maxBatchAverage

	var sb strings.Builder
	for _, avg := range lineSums {
		sb.WriteString(strings.Repeat("*", int(avg*scaleDown)) + "\n")
	}
	return strings.TrimSpace(sb.String())
}
