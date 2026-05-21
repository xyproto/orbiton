package main

import (
	"testing"
)

// makeEditor builds a simple editor pre-populated with the given lines.
func makeEditor(lines []string) *Editor {
	e := NewSimpleEditor(80)
	for i, s := range lines {
		e.SetLine(LineIndex(i), s)
	}
	return e
}

// editorLines returns every line in the editor as a string slice.
func editorLines(e *Editor) []string {
	out := make([]string, e.Len())
	for i := range e.Len() {
		out[i] = e.Line(LineIndex(i))
	}
	return out
}

func TestInsertLineBelowAt(t *testing.T) {
	tests := []struct {
		name     string
		initial  []string
		insertAt int
		want     []string
	}{
		{
			name:     "single line, insert below 0",
			initial:  []string{"alpha"},
			insertAt: 0,
			want:     []string{"alpha", ""},
		},
		{
			name:     "three lines, insert below first",
			initial:  []string{"a", "b", "c"},
			insertAt: 0,
			want:     []string{"a", "", "b", "c"},
		},
		{
			name:     "three lines, insert below middle",
			initial:  []string{"a", "b", "c"},
			insertAt: 1,
			want:     []string{"a", "b", "", "c"},
		},
		{
			name:     "three lines, insert below last",
			initial:  []string{"a", "b", "c"},
			insertAt: 2,
			want:     []string{"a", "b", "c", ""},
		},
		{
			name:     "trailing empty lines are trimmed after insert",
			initial:  []string{"x", "y", ""},
			insertAt: 0,
			want:     []string{"x", "", "y"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := makeEditor(tc.initial)
			e.InsertLineBelowAt(LineIndex(tc.insertAt))
			got := editorLines(e)
			if len(got) != len(tc.want) {
				t.Fatalf("Len() = %d, want %d; lines: %q", len(got), len(tc.want), got)
			}
			for i, w := range tc.want {
				if got[i] != w {
					t.Errorf("line %d = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

func TestInsertLineAbove(t *testing.T) {
	tests := []struct {
		name    string
		initial []string
		cursorY int
		want    []string
	}{
		{
			name:    "insert above first line",
			initial: []string{"alpha"},
			cursorY: 0,
			want:    []string{"", "alpha"},
		},
		{
			name:    "three lines, insert above first",
			initial: []string{"a", "b", "c"},
			cursorY: 0,
			want:    []string{"", "a", "b", "c"},
		},
		{
			name:    "three lines, insert above middle",
			initial: []string{"a", "b", "c"},
			cursorY: 1,
			want:    []string{"a", "", "b", "c"},
		},
		{
			name:    "three lines, insert above last",
			initial: []string{"a", "b", "c"},
			cursorY: 2,
			want:    []string{"a", "b", "", "c"},
		},
		{
			name:    "trailing empty lines are trimmed",
			initial: []string{"x", "y", ""},
			cursorY: 0,
			want:    []string{"", "x", "y"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := makeEditor(tc.initial)
			e.pos.sy = tc.cursorY
			e.InsertLineAbove()
			got := editorLines(e)
			if len(got) != len(tc.want) {
				t.Fatalf("Len() = %d, want %d; lines: %q", len(got), len(tc.want), got)
			}
			for i, w := range tc.want {
				if got[i] != w {
					t.Errorf("line %d = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

// TestInsertLineMapContiguous verifies that e.lines has no gaps after repeated
// insertions -- a gap would cause incorrect Len() and silently wrong output.
func TestInsertLineMapContiguous(t *testing.T) {
	e := makeEditor([]string{"a", "b", "c", "d", "e"})

	// Insert above lines 0, 2, 4 in succession.
	for _, y := range []int{0, 2, 4} {
		e.pos.sy = y
		e.InsertLineAbove()
	}

	n := e.Len()
	for i := range n {
		if _, ok := e.lines[i]; !ok {
			t.Errorf("gap in e.lines at key %d (Len=%d)", i, n)
		}
	}
}
