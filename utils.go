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

// Take the first word and increase it to the next git rebase keyword
func nextGitRebaseKeyword(line string) string {
	cycle1 := []string{"pick", "fixup", "reword", "drop", "edit", "squash", "exec", "break", "label", "reset", "merge"}
	cycle2 := []string{"p", "f", "r", "d", "e", "s", "x", "b", "l", "t", "m"}
	first := strings.Fields(line)[0]
	next := ""
	// Check if the word is in cycle1, then set "next" to the next one in the cycle
	for i, w := range cycle1 {
		if first == w {
			if i+1 < len(cycle1) {
				next = cycle1[i+1]
				break
			} else {
				next = cycle1[0]
			}
		}
	}
	if next == "" {
		// Check if the word is in cycle2, then set "next" to the next one in the cycle
		for i, w := range cycle2 {
			if first == w {
				if i+1 < len(cycle2) {
					next = cycle2[i+1]
					break
				} else {
					next = cycle2[0]
				}
			}
		}
	}
	if next == "" {
		// Return the line as it is, no git rebase keyword found
		return line
	}
	// Return the line with the keyword replaced with the next one in cycle1 or cycle2
	return strings.Replace(line, first, next, 1)
}
