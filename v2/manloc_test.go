package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xyproto/mode"
)

func TestManPageLocationKey(t *testing.T) {
	a := manPageLocationKey("NAME\n     ls - list directory contents\n")
	b := manPageLocationKey("NAME\n     ls - list directory contents\n")
	c := manPageLocationKey("NAME\n     cat - concatenate files\n")
	if a != b {
		t.Errorf("the same contents gave different keys: %q and %q", a, b)
	}
	if a == c {
		t.Error("different contents gave the same key")
	}
	// ":" separates the fields in the location history file
	if strings.Contains(a, ":") {
		t.Errorf("the key must not contain a colon: %q", a)
	}
	if !ShouldKeep(a) {
		t.Errorf("%q should be kept in the location history", a)
	}
}

func TestLocationKeyFor(t *testing.T) {
	e := NewSimpleEditor(80)
	e.InsertStringAndMove(nil, "some contents")
	if key := e.locationKeyFor("/home/user/notes.txt"); key != "/home/user/notes.txt" {
		t.Errorf("regular files should key on the filename, got %q", key)
	}
	e.mode = mode.ManPage
	if key := e.locationKeyFor("/home/user/notes.txt"); key != "/home/user/notes.txt" {
		t.Errorf("a real file misdetected as a man page should still key on the filename, got %q", key)
	}
	key := e.locationKeyFor("/tmp/man.XXXX")
	if !strings.HasPrefix(key, manPageKeyPrefix) {
		t.Errorf("man pages should key on the contents, got %q", key)
	}
	if key != manPageLocationKey(e.String()) {
		t.Error("the key does not match the hash of the contents")
	}
}

// Man page keys have to survive a trip through the location history file
func TestManPageKeyRoundTrip(t *testing.T) {
	key := manPageLocationKey("some man page")
	history := make(LocationHistory)
	history.Set(key, LineNumber(42))
	path := filepath.Join(t.TempDir(), "locations.txt")
	if err := history.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadLocationHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	lineNumber, found := loaded.Get(key)
	if !found {
		t.Fatalf("%q was not loaded back", key)
	}
	if lineNumber != LineNumber(42) {
		t.Errorf("line number: got %d, want 42", lineNumber)
	}
}

func TestProseMode(t *testing.T) {
	for _, m := range []mode.Mode{mode.Markdown, mode.Text, mode.ASCIIDoc, mode.Email, mode.Git} {
		if !proseMode(m) {
			t.Errorf("%s should be prose", m)
		}
	}
	for _, m := range []mode.Mode{mode.Go, mode.C, mode.Shell, mode.Python, mode.JSON, mode.YAML, mode.Config, mode.Ini, mode.ManPage} {
		if proseMode(m) {
			t.Errorf("%s should not be prose", m)
		}
	}
}
