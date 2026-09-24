package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// useClipboard replaces the system clipboard with the given contents for the duration of the test
func useClipboard(t *testing.T, contents string, err error) {
	t.Helper()
	original := readClipboard
	readClipboard = func() (string, error) { return contents, err }
	t.Cleanup(func() { readClipboard = original })
}

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
	useClipboard(t, "", errors.New("no clipboard"))
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

// An empty system clipboard, like on macOS where pbpaste then succeeds, should not clear the internal copy buffer
func TestPasteEmptyClipboard(t *testing.T) {
	useClipboard(t, "", nil)
	undo.Reset()
	e := makeEditor([]string{"first", "", "last"})
	placeCursor(e, 1, 0)
	pasteTwice(t, e, []string{"one", "two", "three"})
	if got := strings.Join(editorLines(e), "\n"); got != "first\none\ntwo\nthree\nlast" {
		t.Errorf("after double paste: %q", got)
	}
}

func TestPasteFromClipboard(t *testing.T) {
	useClipboard(t, "alpha\nbeta", nil)
	undo.Reset()
	e := makeEditor([]string{"first", "", "last"})
	placeCursor(e, 1, 0)
	pasteTwice(t, e, []string{"one", "two", "three"})
	if got := strings.Join(editorLines(e), "\n"); got != "first\nalpha\nbeta\nlast" {
		t.Errorf("after double paste: %q", got)
	}
}
