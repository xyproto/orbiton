package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// TestMarkdownQuoteHighlightPreservesLeadingSpaces verifies that when a line
// like "         |-> c" is highlighted as a Markdown quote (because "> " is
// found near the start of the non-whitespace portion), the leading whitespace
// is preserved in the rendered output.
func TestMarkdownQuoteHighlightPreservesLeadingSpaces(t *testing.T) {
	c := vt.NewCanvasWithSize(80, 10)
	e := NewSimpleEditor(80)
	e.mode = mode.Markdown
	e.syntaxHighlight = true

	// Build the scenario: line 0 is normal text, line 1 has leading spaces + "|-> c"
	e.InsertStringAndMove(c, "a -> b -> c -> d -> e")
	e.ReturnPressed(c, nil)
	e.InsertStringAndMove(c, "         |-> c")

	t.Logf("line 1 data: %q", e.CurrentLine())

	// Render via WriteLines (the path used during a full redraw)
	c = vt.NewCanvasWithSize(80, 10)
	e.WriteLines(c, 0, 2, 0, 0, false, true)
	var buf bytes.Buffer
	if err := c.Snapshot(&buf); err != nil {
		t.Fatal(err)
	}
	snap := buf.String()
	t.Logf("snapshot:\n%s", snap)

	// The snapshot must contain the leading 9 spaces before the pipe
	if !strings.Contains(snap, "         |-> c") {
		t.Errorf("leading whitespace lost in rendered output:\n%s", snap)
	}
}
