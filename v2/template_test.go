package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The names appended to the .gitignore template should be the executables that
// are likely to be built in the directory, without any "v2" style elements.
func TestGitignoreExecutableNames(t *testing.T) {
	base := t.TempDir()

	newDir := func(name, goMod string) string {
		dir := filepath.Join(base, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if goMod != "" {
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}

	for _, tc := range []struct {
		name     string
		goMod    string
		expected []string
	}{
		{"myproj", "module github.com/someone/myproj\n\ngo 1.25\n", []string{"myproj"}},
		{"checkout", "module example.com/a/tool/v3\n", []string{"checkout", "tool"}},
		{"v2", "module github.com/someone/hello/v2\n", []string{"hello"}},
		{"cproject", "", []string{"cproject"}},
	} {
		got := gitignoreExecutableNames(newDir(tc.name, tc.goMod))
		if !slices.Equal(got, tc.expected) {
			t.Errorf("%s: got %v, expected %v", tc.name, got, tc.expected)
		}
	}
}

func TestMajorVersionElement(t *testing.T) {
	for _, s := range []string{"v2", "v10"} {
		if !majorVersionElement(s) {
			t.Errorf("%q should be a major version element", s)
		}
	}
	for _, s := range []string{"", "v", "v2x", "orbiton", "2"} {
		if majorVersionElement(s) {
			t.Errorf("%q should not be a major version element", s)
		}
	}
}
