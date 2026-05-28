package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestBencodeStr verifies that bencodeStr produces the correct length-prefixed
// bencode string encoding.
func TestBencodeStr(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "0:"},
		{"hello", "5:hello"},
		{"op", "2:op"},
		{"eval", "4:eval"},
	}
	for _, tt := range tests {
		if got := string(bencodeStr(tt.input)); got != tt.want {
			t.Errorf("bencodeStr(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestBencodeMsg verifies that bencodeMsg produces a valid bencode dict with
// keys in sorted order.
func TestBencodeMsg(t *testing.T) {
	// Single key
	got := string(bencodeMsg(map[string]string{"op": "clone"}))
	want := "d2:op5:clonee"
	if got != want {
		t.Errorf("bencodeMsg single key: got %q, want %q", got, want)
	}
	// Multiple keys must come out sorted
	got = string(bencodeMsg(map[string]string{
		"op":      "eval",
		"session": "abc",
		"code":    "(+ 1 2)",
	}))
	want = "d4:code7:(+ 1 2)2:op4:eval7:session3:abce"
	if got != want {
		t.Errorf("bencodeMsg sorted keys: got %q, want %q", got, want)
	}
}

// TestBencodeRead verifies that bencodeRead correctly decodes strings, lists
// and dicts from a bencode byte stream.
func TestBencodeRead(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  any
	}{
		{
			name:  "simple string",
			input: "5:hello",
			want:  "hello",
		},
		{
			name:  "empty string",
			input: "0:",
			want:  "",
		},
		{
			name:  "list of strings",
			input: "l4:donee",
			want:  []any{"done"},
		},
		{
			name:  "dict with one key",
			input: "d2:op5:clonee",
			want:  map[string]any{"op": "clone"},
		},
		{
			name:  "dict with status list",
			input: "d7:session3:abc6:statusl4:doneee",
			want:  map[string]any{"session": "abc", "status": []any{"done"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader([]byte(tt.input)))
			got, err := bencodeRead(r)
			if err != nil {
				t.Fatalf("bencodeRead error: %v", err)
			}
			if !bencodeEqual(got, tt.want) {
				t.Errorf("bencodeRead(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// bencodeEqual is a helper for deep-equal comparison of decoded bencode values.
func bencodeEqual(a, b any) bool {
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !bencodeEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !bencodeEqual(v, bv[k]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// TestFindNREPLPort verifies that findNREPLPort locates a .nrepl-port file in
// the given directory and in parent directories.
func TestFindNREPLPort(t *testing.T) {
	// Build a small temp tree:  root/.nrepl-port  and  root/sub/
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	portFile := filepath.Join(root, ".nrepl-port")
	if err := os.WriteFile(portFile, []byte("7888\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Found directly in root
	port, err := findNREPLPort(root)
	if err != nil {
		t.Fatalf("findNREPLPort(root): %v", err)
	}
	if port != 7888 {
		t.Errorf("findNREPLPort(root) = %d, want 7888", port)
	}

	// Found by walking up from sub
	port, err = findNREPLPort(sub)
	if err != nil {
		t.Fatalf("findNREPLPort(sub): %v", err)
	}
	if port != 7888 {
		t.Errorf("findNREPLPort(sub) = %d, want 7888", port)
	}

	// Not found when there is no .nrepl-port anywhere
	empty := t.TempDir()
	if _, err := findNREPLPort(empty); err == nil {
		t.Error("findNREPLPort(empty): expected error, got nil")
	}
}

// TestClojureTopLevelForm verifies that clojureTopLevelForm extracts the
// correct top-level form for several cursor positions and source shapes.
func TestClojureTopLevelForm(t *testing.T) {
	tests := []struct {
		name    string
		source  string // full source text (lines joined by \n)
		cursorY int    // data line the cursor is on
		want    string // expected form text (trailing \n included)
	}{
		{
			name:    "simple defn, cursor on first line",
			source:  "(defn hello []\n  (println \"Hello\"))\n",
			cursorY: 0,
			want:    "(defn hello []\n  (println \"Hello\"))\n",
		},
		{
			name:    "simple defn, cursor inside body",
			source:  "(defn hello []\n  (println \"Hello\"))\n",
			cursorY: 1,
			want:    "(defn hello []\n  (println \"Hello\"))\n",
		},
		{
			name:    "cursor on second form",
			source:  "(def x 1)\n\n(def y 2)\n",
			cursorY: 2,
			want:    "(def y 2)\n",
		},
		{
			name:    "parens inside string do not affect depth",
			source:  "(def msg \"hello (world)\")\n",
			cursorY: 0,
			want:    "(def msg \"hello (world)\")\n",
		},
		{
			name:    "parens inside comment do not affect depth",
			source:  "(defn f []\n  ; closes here: )\n  42)\n",
			cursorY: 0,
			want:    "(defn f []\n  ; closes here: )\n  42)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewSimpleEditor(80)
			e.lines = make(map[int][]rune)
			lines := splitLines(tt.source)
			for i, l := range lines {
				e.lines[i] = []rune(l)
			}
			// Place the cursor at the requested line.
			e.pos.offsetY = 0
			e.pos.sy = tt.cursorY
			e.pos.sx = 0

			got := e.clojureTopLevelForm()
			if got != tt.want {
				t.Errorf("clojureTopLevelForm:\ngot  %q\nwant %q", got, tt.want)
			}
		})
	}
}

// splitLines splits s into a slice of lines without trailing newlines.
func splitLines(s string) []string {
	if len(s) == 0 {
		return nil
	}
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
