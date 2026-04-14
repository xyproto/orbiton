package main

import (
	"bytes"
	"testing"

	"github.com/xyproto/mode"
	"github.com/xyproto/syntax"
)

// originalDefaultTextConfig captures the hardcoded default before any test can mutate it.
var originalDefaultTextConfig = syntax.DefaultTextConfig

// savedDefaultTextConfig saves and restores syntax.DefaultTextConfig around a test,
// so that theme state from other tests (e.g. TestEditor setting zulu) does not leak in.
func savedDefaultTextConfig(t *testing.T) {
	t.Helper()
	saved := syntax.DefaultTextConfig
	t.Cleanup(func() { syntax.DefaultTextConfig = saved })
	syntax.DefaultTextConfig = originalDefaultTextConfig
}

func TestRPMHighlight(t *testing.T) {
	savedDefaultTextConfig(t)
	syntax.AdjustKeywords(mode.Spec)
	input := []byte("%install")
	highlighted, err := syntax.AsText(input, mode.Spec)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<red>%install<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestClojureHighlight(t *testing.T) {
	savedDefaultTextConfig(t)
	syntax.AdjustKeywords(mode.Clojure)
	input := []byte("*agent*")
	highlighted, err := syntax.AsText(input, mode.Clojure)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<red>*agent*<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestC3Highlight(t *testing.T) {
	savedDefaultTextConfig(t)
	syntax.AdjustKeywords(mode.C3)
	input := []byte("$alignof")
	highlighted, err := syntax.AsText(input, mode.C3)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<red>$alignof<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestShellHighlight(t *testing.T) {
	savedDefaultTextConfig(t)
	syntax.AdjustKeywords(mode.Shell)
	input := []byte("--force")
	highlighted, err := syntax.AsText(input, mode.Shell)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<red>--force<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestVibe67Highlight(t *testing.T) {
	savedDefaultTextConfig(t)
	syntax.AdjustKeywords(mode.Vibe67)
	input := []byte("<<<b")
	highlighted, err := syntax.AsText(input, mode.Vibe67)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<red><<<b<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestPrologHighlight(t *testing.T) {
	savedDefaultTextConfig(t)
	// Initialize keywords for Prolog mode
	syntax.AdjustKeywords(mode.Prolog)

	input := []byte("code % comment")
	highlighted, err := syntax.AsText(input, mode.Prolog)
	if err != nil {
		t.Fatal(err)
	}

	// The word "code" should be plaintext (white), and "% comment" should be a comment (darkgray)
	// The highlighter outputs token by token, so we check for several tags.
	if !bytes.Contains(highlighted, []byte("<darkgray>%<off>")) || !bytes.Contains(highlighted, []byte("<darkgray>comment<off>")) {
		t.Errorf("Expected comment to be highlighted, got:\n%q", string(highlighted))
	}
}
