package main

import (
	"testing"
)

// placeCursor moves the editor cursor to the given data line and column,
// without scrolling, for use in tests.
func placeCursor(e *Editor, y, x int) {
	e.pos.sy = y
	e.pos.offsetY = 0
	e.pos.sx = x
	e.pos.offsetX = 0
}

func TestInsertText(t *testing.T) {
	tests := []struct {
		name    string
		initial []string
		y, x    int
		text    string
		want    []string
		wantY   int
		wantX   int
	}{
		{
			name:    "single line into empty line",
			initial: []string{""},
			y:       0, x: 0,
			text:  "hello",
			want:  []string{"hello"},
			wantY: 0, wantX: 5,
		},
		{
			name:    "single line in the middle of a line",
			initial: []string{"axb"},
			y:       0, x: 1,
			text:  "YY",
			want:  []string{"aYYxb"},
			wantY: 0, wantX: 3,
		},
		{
			name:    "two lines splits the current line",
			initial: []string{"axb"},
			y:       0, x: 1,
			text:  "1\n2",
			want:  []string{"a1", "2xb"},
			wantY: 1, wantX: 1,
		},
		{
			name:    "multiple lines with a middle line",
			initial: []string{"start", "after"},
			y:       0, x: 5,
			text:  "A\nB\nC",
			want:  []string{"startA", "B", "C", "after"},
			wantY: 2, wantX: 1,
		},
		{
			name:    "carriage returns are normalized",
			initial: []string{""},
			y:       0, x: 0,
			text:  "a\r\nb\rc",
			want:  []string{"a", "b", "c"},
			wantY: 2, wantX: 1,
		},
		{
			name:    "non-breaking space becomes a regular space",
			initial: []string{""},
			y:       0, x: 0,
			text:  "a\u00A0b",
			want:  []string{"a b"},
			wantY: 0, wantX: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := makeEditor(tc.initial)
			placeCursor(e, tc.y, tc.x)
			e.InsertText(nil, tc.text)

			got := editorLines(e)
			if len(got) != len(tc.want) {
				t.Fatalf("Len() = %d, want %d; lines: %q", len(got), len(tc.want), got)
			}
			for i, w := range tc.want {
				if got[i] != w {
					t.Errorf("line %d = %q, want %q", i, got[i], w)
				}
			}
			if gotY := int(e.DataY()); gotY != tc.wantY {
				t.Errorf("cursor Y = %d, want %d", gotY, tc.wantY)
			}
			// DataX returns the rune count plus a sentinel error when the
			// cursor sits right after the line contents, which is a valid
			// paste position, so only the column count is checked here.
			if gotX, _ := e.DataX(); gotX != tc.wantX {
				t.Errorf("cursor X = %d, want %d", gotX, tc.wantX)
			}
		})
	}
}
