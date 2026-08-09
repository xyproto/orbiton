package main

import "testing"

func editorWithLines(lines ...string) *Editor {
	e := NewSimpleEditor(80)
	for i, line := range lines {
		e.lines[i] = []rune(line)
	}
	return e
}

func TestFirstConflictMarker(t *testing.T) {
	e := editorWithLines(
		"package main",
		"",
		"<<<<<<< HEAD",
		"\tfmt.Println(\"ours\")",
		"=======",
		"\tfmt.Println(\"theirs\")",
		">>>>>>> feature",
	)
	line, found := e.FirstConflictMarker()
	if !found {
		t.Fatal("the conflict marker was not found")
	}
	if line != LineIndex(2) {
		t.Errorf("line index: got %d, want 2", line)
	}
}

func TestNoConflictMarker(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{"plain code", []string{"package main", "", "func main() {}"}},
		{"no trailing space", []string{"<<<<<<<HEAD"}},
		{"too few angles", []string{"<<<<<< HEAD"}},
		{"not at the start of the line", []string{"// <<<<<<< HEAD"}},
		{"only the end marker", []string{">>>>>>> feature"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, found := editorWithLines(tc.lines...).FirstConflictMarker(); found {
				t.Error("a conflict marker was found where there is none")
			}
		})
	}
}
