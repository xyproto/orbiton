package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xyproto/mode"
	"github.com/xyproto/syntax"
	"github.com/xyproto/themes"
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

func TestTimHighlight(t *testing.T) {
	adjustSyntaxHighlightingKeywords(mode.Tim)
	input := []byte("<<<b")
	highlighted, err := AsText(input, mode.Tim)
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

func TestFishAndOilHighlight(t *testing.T) {
	for _, m := range []mode.Mode{mode.Fish, mode.Oil} {
		adjustSyntaxHighlightingKeywords(m)
		highlighted, err := AsText([]byte("--force"), m)
		if err != nil {
			t.Fatal(err)
		}
		expected := []byte("<lightred>--force<off>")
		if !bytes.Equal(highlighted, expected) {
			t.Errorf("Expected %q for %s, got %q", string(expected), m, string(highlighted))
		}
	}
}

func TestVSLightNoTagLeak(t *testing.T) {
	registerVSColors()
	theme := themes.NewVSTrueColorTheme()
	syntax.DefaultTextConfig = *(theme.TextConfig())
	highlighted, err := AsText([]byte("hello world"), mode.Go)
	if err != nil {
		t.Fatal(err)
	}
	rendered := tout.DarkTags(string(highlighted))
	if strings.Contains(rendered, "<vsplain>") {
		t.Errorf("Tag <vsplain> was leaked into rendered output: %q", rendered)
	}
	if strings.Contains(rendered, "<") && strings.Contains(rendered, ">") {
		t.Errorf("Unexpanded tags leaked into rendered output: %q", rendered)
	}
}

func TestDirectHighlighting(t *testing.T) {
	var buf bytes.Buffer
	err := syntax.Highlight([]byte("func main() {}"), mode.Go, syntax.VSLightTheme, &buf)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "<") && strings.Contains(out, ">") {
		t.Errorf("Tags leaked into direct highlighted output: %q", out)
	}
	attrs := syntax.HighlightToAttributes([]byte("func main() {}"), mode.Go, syntax.VSLightTheme)
	if len(attrs) == 0 {
		t.Errorf("Expected attributes, got none")
	}
}
