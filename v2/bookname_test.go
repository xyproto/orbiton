package main

import "testing"

func TestIsBookFilename(t *testing.T) {
	for name, want := range map[string]bool{
		"book.md": true, "Book2.md": true, "/home/x/BOOK-draft.MD": true, "books/notes.md": false,
		"mybook.md": false, "book.txt": false, "book": false,
	} {
		if got := isBookFilename(name); got != want {
			t.Errorf("isBookFilename(%q) = %v, want %v", name, got, want)
		}
	}
}
