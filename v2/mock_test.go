package main

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// TestMockTTY_ReadsScriptedKeys verifies that vt.NewTTYFromReader lets us
// feed an orbiton-facing *vt.TTY from an in-memory byte stream, with no
// real terminal required.
func TestMockTTY_ReadsScriptedKeys(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mock TTY key reading is not supported on Windows")
	}
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

// TestMockTTY_NavigationKeys feeds escape sequences for the common navigation
// keys into a mock TTY and verifies that ReadKey decodes each one correctly.
func TestMockTTY_NavigationKeys(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mock TTY key reading is not supported on Windows")
	}
	// Build a script of raw terminal bytes for: Up, Down, Left, Right, Page
	// Up, Page Down, Delete, F1, F2.
	script := []byte{
		27, 91, 65, // Up Arrow    (ESC [ A)
		27, 91, 66, // Down Arrow  (ESC [ B)
		27, 91, 68, // Left Arrow  (ESC [ D)
		27, 91, 67, // Right Arrow (ESC [ C)
		27, 91, 53, 126, // Page Up     (ESC [ 5 ~)
		27, 91, 54, 126, // Page Down   (ESC [ 6 ~)
		27, 91, 51, 126, // Delete      (ESC [ 3 ~)
		27, 79, 80, // F1          (ESC O P)
		27, 79, 81, // F2          (ESC O Q)
	}
	tty := vt.NewTTYFromReader(bytes.NewReader(script))
	defer tty.Close()

	want := []string{"↑", "↓", "←", "→", "⇞", "⇟", "⌦", "F1", "F2"}
	for i, w := range want {
		if got := tty.ReadKey(); got != w {
			t.Errorf("key %d: got %q, want %q", i, got, w)
		}
	}
}

// TestMockTTY_ModifierArrowKeys verifies that modifier+arrow CSI sequences
// (Ctrl, Alt, Shift combined with arrow keys) are decoded to the correct
// strings by a mock TTY.
func TestMockTTY_ModifierArrowKeys(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mock TTY key reading is not supported on Windows")
	}
	script := []byte{
		27, 91, 49, 59, 53, 65, // Ctrl-Up    (ESC [ 1 ; 5 A)
		27, 91, 49, 59, 51, 66, // Alt-Down   (ESC [ 1 ; 3 B)
		27, 91, 49, 59, 50, 68, // Shift-Left (ESC [ 1 ; 2 D)
		27, 91, 49, 59, 53, 67, // Ctrl-Right (ESC [ 1 ; 5 C)
	}
	tty := vt.NewTTYFromReader(bytes.NewReader(script))
	defer tty.Close()

	want := []string{"ctrl↑", "alt↓", "shift←", "ctrl→"}
	for i, w := range want {
		if got := tty.ReadKey(); got != w {
			t.Errorf("key %d: got %q, want %q", i, got, w)
		}
	}
}

// TestMockCanvas_DeleteRestOfLine inserts "hello world", moves the cursor to
// the space after "hello", and calls DeleteRestOfLine. Only "hello" should
// remain on the line.
func TestMockCanvas_DeleteRestOfLine(t *testing.T) {
	c := vt.NewCanvasWithSize(80, 5)
	e := NewSimpleEditor(80)

	e.InsertString(c, "hello world")
	e.Home()
	for range 5 {
		e.Next(c)
	}
	e.DeleteRestOfLine()

	if got := e.Line(0); got != "hello" {
		t.Errorf("DeleteRestOfLine: got %q, want %q", got, "hello")
	}
}

// TestMockCanvas_UndoRestore takes an undo snapshot of a fresh editor,
// inserts text, then restores. The editor must return to its pre-insert state.
func TestMockCanvas_UndoRestore(t *testing.T) {
	e := NewSimpleEditor(80)
	u := NewUndo(64, 1024*1024)

	u.Snapshot(e)

	e.InsertStringAndMove(nil, "hello")
	if got := e.String(); got != "hello\n" {
		t.Fatalf("before restore: got %q, want %q", got, "hello\n")
	}

	if err := u.Restore(e); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := e.String(); got != "" {
		t.Errorf("after restore: got %q, want empty string", got)
	}
}

// TestMockCanvas_JoinLines inserts two lines then joins them with
// JoinLineWithNext. The result must be a single line containing both words
// separated by a space.
func TestMockCanvas_JoinLines(t *testing.T) {
	c := vt.NewCanvasWithSize(80, 5)
	e := NewSimpleEditor(80)

	e.InsertStringAndMove(c, "foo")
	e.ReturnPressed(c, nil, false)
	e.InsertStringAndMove(c, "bar")

	e.Up(c, nil)
	e.JoinLineWithNext(c)

	if got := e.Line(0); got != "foo bar" {
		t.Errorf("JoinLineWithNext: got %q, want %q", got, "foo bar")
	}
	if n := e.Len(); n != 1 {
		t.Errorf("JoinLineWithNext: Len = %d, want 1", n)
	}
}

// TestMockCanvas_CommentToggle checks that CommentOn prepends "// " and
// CommentOff removes it, leaving the original line intact.
func TestMockCanvas_CommentToggle(t *testing.T) {
	e := NewSimpleEditor(0)
	e.mode = mode.Go

	e.InsertStringAndMove(nil, "x := 1")
	e.Home()

	e.CommentOn("//")
	if got := e.CurrentLine(); !strings.HasPrefix(got, "// ") {
		t.Errorf("CommentOn: got %q, want prefix %q", got, "// ")
	}

	e.CommentOff("//")
	if got := e.CurrentLine(); got != "x := 1" {
		t.Errorf("CommentOff: got %q, want %q", got, "x := 1")
	}
}

// TestMockCanvas_WordCount inserts a known sentence and checks that WordCount
// returns the expected number of words.
func TestMockCanvas_WordCount(t *testing.T) {
	e := NewSimpleEditor(0)
	e.InsertStringAndMove(nil, "one two three")
	if got := e.WordCount(); got != 3 {
		t.Errorf("WordCount: got %d, want 3", got)
	}
}

// TestMockCanvas_ReturnCreatesNewLine verifies that ReturnPressed splits the
// document into two lines and that WriteLines renders both onto the canvas.
func TestMockCanvas_ReturnCreatesNewLine(t *testing.T) {
	c := vt.NewCanvasWithSize(80, 5)
	e := NewSimpleEditor(80)

	e.InsertStringAndMove(c, "first")
	e.ReturnPressed(c, nil, false)
	e.InsertStringAndMove(c, "second")

	if n := e.Len(); n != 2 {
		t.Fatalf("Len = %d, want 2", n)
	}
	if got := e.Line(0); got != "first" {
		t.Errorf("line 0: got %q, want %q", got, "first")
	}
	if got := e.Line(1); got != "second" {
		t.Errorf("line 1: got %q, want %q", got, "second")
	}

	// Render both lines onto the mock canvas and confirm they appear in the snapshot.
	e.WriteLines(c, 0, 2, 0, 0, false, true)
	var buf bytes.Buffer
	if err := c.Snapshot(&buf); err != nil {
		t.Fatal(err)
	}
	snap := buf.String()
	if !strings.Contains(snap, "first") {
		t.Errorf("snapshot missing 'first':\n%s", snap)
	}
	if !strings.Contains(snap, "second") {
		t.Errorf("snapshot missing 'second':\n%s", snap)
	}
}

// TestMockCanvas_GoToNextWord verifies that GoToNextWord advances the cursor
// from the start of "foo bar" to the start of "bar" (data position 4).
func TestMockCanvas_GoToNextWord(t *testing.T) {
	c := vt.NewCanvasWithSize(80, 5)
	e := NewSimpleEditor(80)

	e.InsertString(c, "foo bar")
	e.Home()

	e.GoToNextWord(c, nil)

	x, err := e.DataX()
	if err != nil {
		t.Fatalf("DataX after GoToNextWord: %v", err)
	}
	if x != 4 {
		t.Errorf("DataX after GoToNextWord: got %d, want 4", x)
	}
}
