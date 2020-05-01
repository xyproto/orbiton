package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// exists checks if the given path exists
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// which tries to find the given executable name in the $PATH
// Returns an empty string if not found.
func which(executable string) string {
	p, err := exec.LookPath(executable)
	if err != nil {
		return ""
	}
	return p
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

// logf, for quick "printf-style" debugging
func logf(head string, tail ...interface{}) {
	tmpdir := os.Getenv("TMPDIR")
	if tmpdir == "" {
		tmpdir = "/tmp"
	}
	logfilename := filepath.Join(tmpdir, "o.log")
	f, err := os.OpenFile(logfilename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		f, err = os.Create(logfilename)
		if err != nil {
			log.Fatalln(err)
		}
	}
	f.WriteString(fmt.Sprintf(head, tail...))
	f.Sync()
	f.Close()
}

// Silence the "logf is unused" message by staticcheck
var _ = logf
