package main

import (
	"bytes"
	"testing"

	"github.com/xyproto/mode"
)

func TestRPMHighlight(t *testing.T) {
	adjustSyntaxHighlightingKeywords(mode.Spec)
	input := []byte("%install")
	highlighted, err := AsText(input, mode.Spec)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<lightred>%install<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestClojureHighlight(t *testing.T) {
	adjustSyntaxHighlightingKeywords(mode.Clojure)
	input := []byte("*agent*")
	highlighted, err := AsText(input, mode.Clojure)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<lightred>*agent*<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestC3Highlight(t *testing.T) {
	adjustSyntaxHighlightingKeywords(mode.C3)
	input := []byte("$alignof")
	highlighted, err := AsText(input, mode.C3)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<lightred>$alignof<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestShellHighlight(t *testing.T) {
	adjustSyntaxHighlightingKeywords(mode.Shell)
	input := []byte("--force")
	highlighted, err := AsText(input, mode.Shell)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<lightred>--force<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestVibe67Highlight(t *testing.T) {
	adjustSyntaxHighlightingKeywords(mode.Vibe67)
	input := []byte("<<<b")
	highlighted, err := AsText(input, mode.Vibe67)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("<lightred><<<b<off>")
	if !bytes.Equal(highlighted, expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(highlighted))
	}
}

func TestPrologHighlight(t *testing.T) {
	// Initialize keywords for Prolog mode
	adjustSyntaxHighlightingKeywords(mode.Prolog)

	input := []byte("code % comment")
	highlighted, err := AsText(input, mode.Prolog)
	if err != nil {
		t.Fatal(err)
	}

	// The word "code" should be plaintext (white), and "% comment" should be a comment (darkgray)
	// The highlighter outputs token by token, so we check for several tags.
	if !bytes.Contains(highlighted, []byte("<gray>%<off>")) || !bytes.Contains(highlighted, []byte("<gray>comment<off>")) {
		t.Errorf("Expected comment to be highlighted, got:\n%q", string(highlighted))
	}
}
