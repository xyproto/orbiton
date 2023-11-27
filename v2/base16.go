package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/xyproto/env/v2"
)

// TerminalColor represents a terminal color with its RGB components.
type TerminalColor struct {
	R, G, B int
}

// Define the 16 standard terminal colors.
var terminalColors = []TerminalColor{
	{0, 0, 0},       // Black
	{128, 0, 0},     // Red
	{0, 128, 0},     // Green
	{128, 128, 0},   // Yellow
	{0, 0, 128},     // Blue
	{128, 0, 128},   // Magenta
	{0, 128, 128},   // Cyan
	{192, 192, 192}, // Light gray
	{128, 128, 128}, // Dark gray
	{255, 0, 0},     // Light red
	{0, 255, 0},     // Light green
	{255, 255, 0},   // Light yellow
	{0, 0, 255},     // Light blue
	{255, 0, 255},   // Light magenta
	{0, 255, 255},   // Light cyan
	{255, 255, 255}, // White
}

// hexToRGB converts a hex color string to its RGB components.
func hexToRGB(hex string) (int, int, int, error) {
	r, err := strconv.ParseInt(hex[0:2], 16, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	g, err := strconv.ParseInt(hex[2:4], 16, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	b, err := strconv.ParseInt(hex[4:6], 16, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	return int(r), int(g), int(b), nil
}

// colorDistance calculates the Euclidean distance between two colors.
func colorDistance(c1, c2 TerminalColor) float64 {
	return math.Sqrt(float64((c1.R-c2.R)*(c1.R-c2.R) + (c1.G-c2.G)*(c1.G-c2.G) + (c1.B-c2.B)*(c1.B-c2.B)))
}

// closestTerminalColor finds the closest terminal color to the given RGB color.
func closestTerminalColor(r, g, b int) int {
	minDistance := math.MaxFloat64
	closestIndex := 0

	for i, color := range terminalColors {
		distance := colorDistance(TerminalColor{r, g, b}, color)
		if distance < minDistance {
			minDistance = distance
			closestIndex = i
		}
	}

	return closestIndex
}

// ConvertHexToTerminalColor converts a hex color to the closest terminal color index.
func ConvertHexToTerminalColor(hex string) (int, error) {
	r, g, b, err := hexToRGB(hex)
	if err != nil {
		return 0, err
	}
	return closestTerminalColor(r, g, b), nil
}

// Base16 tries to fetch terminal colors from a Base16 color scheme
func Base16() {
	hexColors := []string{
		env.Str("BASE16_COLOR_00_HEX", "272822"),
		env.Str("BASE16_COLOR_01_HEX", "383830"),
		env.Str("BASE16_COLOR_02_HEX", "49483e"),
		env.Str("BASE16_COLOR_03_HEX", "75715e"),
		env.Str("BASE16_COLOR_04_HEX", "a59f85"),
		env.Str("BASE16_COLOR_05_HEX", "f8f8f2"),
		env.Str("BASE16_COLOR_06_HEX", "f5f4f1"),
		env.Str("BASE16_COLOR_07_HEX", "f9f8f5"),
		env.Str("BASE16_COLOR_08_HEX", "f92672"),
		env.Str("BASE16_COLOR_09_HEX", "fd971f"),
		env.Str("BASE16_COLOR_0A_HEX", "f4bf75"),
		env.Str("BASE16_COLOR_0B_HEX", "a6e22e"),
		env.Str("BASE16_COLOR_0C_HEX", "a1efe4"),
		env.Str("BASE16_COLOR_0D_HEX", "66d9ef"),
		env.Str("BASE16_COLOR_0E_HEX", "ae81ff"),
		env.Str("BASE16_COLOR_0F_HEX", "cc6633"),
	}
	for _, hexColor := range hexColors {
		terminalColorIndex, err := ConvertHexToTerminalColor(hexColor)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Printf("Closest terminal color for #%s is %d\n", hexColor, terminalColorIndex)
	}
}
