package main

import (
	"strings"
	"testing"
	"time"

	"github.com/xyproto/clip"
)

// pasteTwice simulates pressing ctrl-v twice on the same line, with the given lines in the internal copy buffer
func pasteTwice(t *testing.T, e *Editor, lines []string) {
	t.Helper()
	status := e.NewStatusBar(time.Second, "")
	copyLines := lines
	previousCopyLines := lines
	firstPasteAction := false
	e.Paste(nil, status, &copyLines, &previousCopyLines, &firstPasteAction, false, false)
	e.Paste(nil, status, &copyLines, &previousCopyLines, &firstPasteAction, false, true)
}

func TestUndoDoublePaste(t *testing.T) {
	if s, err := clip.ReadAll(false); err == nil && strings.TrimSpace(s) != "" {
		t.Skip("the system clipboard is available and would be used instead of the internal copy buffer")
	}
	undo.Reset()
	e := makeEditor([]string{"first", "", "last"})
	placeCursor(e, 1, 0)
	before := strings.Join(editorLines(e), "\n")
	pasteTwice(t, e, []string{"one", "two", "three"})
	if got := strings.Join(editorLines(e), "\n"); got != "first\none\ntwo\nthree\nlast" {
		t.Fatalf("after double paste: %q", got)
	}
	if undo.Len() != 1 {
		t.Fatalf("expected 1 undo snapshot after a double paste, got %d", undo.Len())
	}
	if err := undo.Restore(e); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(editorLines(e), "\n"); got != before {
		t.Errorf("after undo: %q, want %q", got, before)
	}
	if e.lastPasteY != -1 {
		t.Errorf("lastPasteY = %d after undo, want -1", e.lastPasteY)
	}
}
