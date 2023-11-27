package main

import (
	"testing"
)

// TestConvertHexToTerminalColor tests the ConvertHexToTerminalColor function.
func TestConvertHexToTerminalColor(t *testing.T) {
	// Define test cases
	testCases := []struct {
		hex      string
		expected int
	}{
		{"000000", 0},  // Black
		{"800000", 1},  // Red
		{"008000", 2},  // Green
		{"808000", 3},  // Yellow
		{"000080", 4},  // Blue
		{"800080", 5},  // Magenta
		{"008080", 6},  // Cyan
		{"C0C0C0", 7},  // Light gray
		{"808080", 8},  // Dark gray
		{"FF0000", 9},  // Light red
		{"00FF00", 10}, // Light green
		{"FFFF00", 11}, // Light yellow
		{"0000FF", 12}, // Light blue
		{"FF00FF", 13}, // Light magenta
		{"00FFFF", 14}, // Light cyan
		{"FFFFFF", 15}, // White
		// Additional test cases
		{"272822", 0},  // Close to black
		{"f92672", 9},  // Close to light red
		{"a6e22e", 11}, // Close to light green
		{"66d9ef", 7},  // Close to light blue
		{"ae81ff", 7},  // Close to light magenta
		{"f4bf75", 7},  // Close to light yellow
		{"cc6633", 3},  // Close to light red
	}

	// Iterate over test cases
	for _, tc := range testCases {
		t.Run("Hex:"+tc.hex, func(t *testing.T) {
			got, err := ConvertHexToTerminalColor(tc.hex)
			if err != nil {
				t.Errorf("ConvertHexToTerminalColor(%s) returned an error: %v", tc.hex, err)
			}
			if got != tc.expected {
				t.Errorf("ConvertHexToTerminalColor(%s) = %d, want %d", tc.hex, got, tc.expected)
			}
		})
	}
}

// TestHexToRGB tests the hexToRGB function.
func TestHexToRGB(t *testing.T) {
	// Define test cases
	testCases := []struct {
		hex      string
		expected [3]int
	}{
		{"000000", [3]int{0, 0, 0}},
		{"FFFFFF", [3]int{255, 255, 255}},
		{"FF0000", [3]int{255, 0, 0}},
		{"00FF00", [3]int{0, 255, 0}},
		{"0000FF", [3]int{0, 0, 255}},
		{"FFFF00", [3]int{255, 255, 0}},
		{"00FFFF", [3]int{0, 255, 255}},
		{"FF00FF", [3]int{255, 0, 255}},
		// Additional test cases
		{"C0C0C0", [3]int{192, 192, 192}},
		{"808080", [3]int{128, 128, 128}},
		{"800000", [3]int{128, 0, 0}},
		{"008000", [3]int{0, 128, 0}},
		{"000080", [3]int{0, 0, 128}},
		{"808000", [3]int{128, 128, 0}},
		{"800080", [3]int{128, 0, 128}},
		{"008080", [3]int{0, 128, 128}},
	}

	// Iterate over test cases
	for _, tc := range testCases {
		t.Run("Hex:"+tc.hex, func(t *testing.T) {
			r, g, b, err := hexToRGB(tc.hex)
			if err != nil {
				t.Errorf("hexToRGB(%s) returned an error: %v", tc.hex, err)
			}
			if r != tc.expected[0] || g != tc.expected[1] || b != tc.expected[2] {
				t.Errorf("hexToRGB(%s) = %d, %d, %d, want %d, %d, %d", tc.hex, r, g, b, tc.expected[0], tc.expected[1], tc.expected[2])
			}
		})
	}
}
