package main

import (
	"fmt"
	"os"
	"strings"
)

// exists checks if the given path exists
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// errLog outputs a message to stderr
func errLog(s string) {
	fmt.Fprintf(os.Stderr, "%s\n", s)
}

func hasAnyPrefixWord(line string, wordList []string) bool {
	for _, word := range wordList {
		if strings.HasPrefix(line, word+" ") {
			return true
		}
	}
	return false
}
