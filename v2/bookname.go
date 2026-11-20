package main

import (
	"path/filepath"
	"strings"
)

// isBookFilename checks if the given filename is a Markdown file that starts with "book", like book.md or Book2.md
func isBookFilename(filename string) bool {
	base := strings.ToLower(filepath.Base(filename))
	return strings.HasPrefix(base, "book") && strings.HasSuffix(base, ".md")
}
