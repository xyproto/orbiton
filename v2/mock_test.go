package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xyproto/vt"
)

// TestMockTTY_ReadsScriptedKeys verifies that vt.NewTTYFromReader lets us
// feed an orbiton-facing *vt.TTY from an in-memory byte stream, with no
// real terminal required.
func TestMockTTY_ReadsScriptedKeys(t *testing.T) {
	// "ab" then ESC-[A (Up arrow) then Ctrl-Q (0x11).
	script := []byte{'a', 'b', 27, '[', 'A', 0x11}
	tty := vt.NewTTYFromReader(bytes.NewReader(script))
	defer tty.Close()

	want := []string{"a", "b", "↑", "c:17"}
	for i, w := range want {
		if got := tty.ReadKey(); got != w {
			t.Errorf("key %d: got %q, want %q", i, got, w)
		}
	}
}

// TestMockCanvas_EditorSnapshot uses vt.NewCanvasWithSize + Canvas.Snapshot
// to drive a deterministic render of an Editor, without touching a real
// terminal. This is the pattern a future testvt/ harness will build on.
func TestMockCanvas_EditorSnapshot(t *testing.T) {
	c := vt.NewCanvasWithSize(20, 3)
	e := NewSimpleEditor(80)

	e.InsertString(c, "hello")
	e.InsertStringBelow(0, "world")

	// Draw the first few lines onto the mock canvas.
	e.WriteLines(c, 0, 3, 0, 0, false, true)

	var buf bytes.Buffer
	if err := c.Snapshot(&buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	// We don't pin the exact contents (the editor may pad or decorate),
	// just confirm the header + our inserted text are present.
	if !strings.HasPrefix(got, "vt-snapshot 1 w=20 h=3\n") {
		t.Errorf("snapshot missing expected header:\n%s", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("snapshot missing 'hello':\n%s", got)
	}
	if !strings.Contains(got, "world") {
		t.Errorf("snapshot missing 'world':\n%s", got)
	}
}
