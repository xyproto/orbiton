package main

import (
	"strings"
	"testing"
)

// TestToggleCheckboxUndo checks that one ctrl-z undoes a checkbox toggle.
func TestToggleCheckboxUndo(t *testing.T) {
	undo.Reset()
	e := makeEditor([]string{"# notes", "", "- [ ] task one", "- [ ] task two"})
	placeCursor(e, 2, 0)
	before := strings.Join(editorLines(e), "\n")

	if !e.ToggleCheckboxCurrentLine() {
		t.Fatal("expected the checkbox on the current line to be toggled")
	}
	if got := e.Line(2); got != "- [x] task one" {
		t.Fatalf("after the toggle: %q", got)
	}
	if undo.Len() != 1 {
		t.Fatalf("expected 1 undo snapshot after a toggle, got %d", undo.Len())
	}
	if err := undo.Restore(e); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(editorLines(e), "\n"); got != before {
		t.Errorf("after undo: %q, want %q", got, before)
	}

	// A line without a checkbox must not take a snapshot
	undo.Reset()
	placeCursor(e, 0, 0)
	if e.ToggleCheckboxCurrentLine() {
		t.Error("expected no checkbox to be toggled on a heading line")
	}
	if undo.Len() != 0 {
		t.Errorf("expected no undo snapshot when nothing was toggled, got %d", undo.Len())
	}
}
