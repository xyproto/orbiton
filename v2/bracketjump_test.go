package main

import (
	"testing"
	"time"

	"github.com/xyproto/vt"
)

func TestJumpToMatchingBracket(t *testing.T) {
	lines := []string{
		"func main() {",
		"    x := f(1, [2])",
		"}",
		"oops(",
	}
	tests := []struct {
		name        string
		y, x        int
		wantHandled bool
		wantY       LineIndex
		wantRune    rune
		wantUnmoved bool
	}{
		{"parenthesis", 0, 9, true, 0, ')', false},
		{"curly bracket across lines", 0, 12, true, 2, '}', false},
		{"closing curly bracket", 2, 0, true, 0, '{', false},
		{"square bracket", 1, 14, true, 1, ']', false},
		{"closing square bracket", 1, 16, true, 1, '[', false},
		{"not on a bracket", 0, 2, false, 0, 'n', true},
		{"no match", 3, 4, true, 3, '(', true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := makeEditor(lines)
			c := vt.NewCanvasWithSize(80, 40)
			status := e.NewStatusBar(time.Second, "")
			placeCursor(e, tc.y, tc.x)
			if handled := e.JumpToMatchingBracket(c, status); handled != tc.wantHandled {
				t.Fatalf("expected handled to be %v, got %v", tc.wantHandled, handled)
			}
			if got := e.DataY(); got != tc.wantY {
				t.Errorf("expected line %d, got %d", tc.wantY, got)
			}
			if got := e.Rune(); got != tc.wantRune {
				t.Errorf("expected the cursor on %q, got %q", tc.wantRune, got)
			}
			if tc.wantUnmoved && e.pos.sx != tc.x {
				t.Errorf("expected the cursor to stay at x=%d, got x=%d", tc.x, e.pos.sx)
			}
		})
	}
}
