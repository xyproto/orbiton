package main

import (
	"testing"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// In book mode, plain Return at the end of a paragraph inserts an extra
// blank line so the cursor lands on a fresh paragraph. Soft-Return
// (shift-Return / alt-Return) suppresses that extra line so the cursor
// drops to the very next line, like shift-Return in a word processor.
func TestSoftReturnInBookMode_NoExtraBlankLine(t *testing.T) {
	c := vt.NewCanvasWithSize(80, 10)

	hardEditor := NewSimpleEditor(80)
	hardEditor.mode = mode.Markdown
	hardEditor.bookMode.Store(true)
	hardEditor.InsertStringAndMove(c, "first line")
	hardEditor.End(c)
	hardEditor.ReturnPressed(c, nil, false)
	hardEditor.InsertStringAndMove(c, "after")

	softEditor := NewSimpleEditor(80)
	softEditor.mode = mode.Markdown
	softEditor.bookMode.Store(true)
	softEditor.InsertStringAndMove(c, "first line")
	softEditor.End(c)
	softEditor.ReturnPressed(c, nil, true)
	softEditor.InsertStringAndMove(c, "after")

	hard := hardEditor.String()
	soft := softEditor.String()
	if hard == soft {
		t.Fatalf("hard and soft return produced identical output: %q", hard)
	}
	want := "first line\nafter\n"
	if soft != want {
		t.Errorf("soft return: got %q, want %q", soft, want)
	}
}

// Soft-Return must not auto-continue a Markdown list prefix in book mode.
// Plain Return inserts a fresh "* " on the next line; soft-Return drops
// the cursor onto a clean line so the user can write continuation text
// inside the same list item.
func TestSoftReturnInBookMode_SkipsListPrefix(t *testing.T) {
	c := vt.NewCanvasWithSize(80, 10)
	e := NewSimpleEditor(80)
	e.mode = mode.Markdown
	e.bookMode.Store(true)
	e.InsertStringAndMove(c, "* item")
	e.End(c)
	e.ReturnPressed(c, nil, true)
	e.InsertStringAndMove(c, "x")

	got := e.String()
	want := "* item\nx\n"
	if got != want {
		t.Errorf("soft return in list: got %q, want %q", got, want)
	}
}
