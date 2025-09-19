package main

import (
	"testing"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// TestTryLSPCompletionGoFile tests that LSP completion is attempted for Go files
func TestTryLSPCompletionGoFile(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Go
	e.filename = "test.go"

	c := vt.NewCanvas()
	result := e.TryLSPCompletion(c)

	if !result {
		t.Error("Expected LSP completion to be attempted for Go files")
	}

	// Check if TEST_COMPLETION was inserted
	content := e.String()
	expected := "TEST_COMPLETION\n"
	if content != expected {
		t.Errorf("Expected %q to be inserted, got %q", expected, content)
	}
}

// TestTryLSPCompletionNonGoFile tests that LSP completion is skipped for non-Go files
func TestTryLSPCompletionNonGoFile(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.C // Not Go

	c := vt.NewCanvas()
	result := e.TryLSPCompletion(c)

	if result {
		t.Error("Expected LSP completion to be skipped for non-Go files")
	}
}
