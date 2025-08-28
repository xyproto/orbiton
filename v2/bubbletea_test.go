package main

import (
	"fmt"
	"os"
	"testing"
)

// TestBubbleteaIntegration tests basic bubbletea integration
func TestBubbleteaIntegration(t *testing.T) {
	// Create a temporary test file
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	tmpFile, err := os.CreateTemp("", "orbiton_test_*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test model creation
	tty, err := NewTTY()
	if err != nil {
		t.Skipf("Cannot create TTY for testing: %v", err)
	}
	defer tty.Close()

	fnord := FilenameOrData{filename: tmpFile.Name()}
	theme := NewDefaultTheme()

	model, err := NewOrbitonModel(
		tty, fnord, 0, 0, false, theme,
		true, false, false, false, false, false, false,
	)

	if err != nil {
		t.Errorf("Failed to create OrbitonModel: %v", err)
		return
	}

	if model == nil {
		t.Error("OrbitonModel is nil")
		return
	}

	if model.editor == nil {
		t.Error("Editor is nil")
		return
	}

	if model.canvas == nil {
		t.Error("Canvas is nil")
		return
	}

	if model.status == nil {
		t.Error("Status is nil")
		return
	}

	fmt.Println("✓ Bubbletea integration basic test passed")
}

// TestEnvironmentVariableControl tests the environment variable control
func TestEnvironmentVariableControl(t *testing.T) {
	// Test default behavior (should be false for compatibility)
	if shouldUseBubbletea() {
		t.Error("shouldUseBubbletea() should default to false")
	}

	// Test explicit enable
	os.Setenv("ORBITON_USE_BUBBLETEA", "1")
	if !shouldUseBubbletea() {
		t.Error("ORBITON_USE_BUBBLETEA=1 should enable bubbletea")
	}
	os.Unsetenv("ORBITON_USE_BUBBLETEA")

	// Test explicit disable
	os.Setenv("ORBITON_NO_BUBBLETEA", "1")
	if shouldUseBubbletea() {
		t.Error("ORBITON_NO_BUBBLETEA=1 should disable bubbletea")
	}
	os.Unsetenv("ORBITON_NO_BUBBLETEA")

	fmt.Println("✓ Environment variable control test passed")
}
