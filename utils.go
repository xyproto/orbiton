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

// hasE checks if the given environment variable is set
func hasE(envVar string) bool {
	return os.Getenv(envVar) != ""
}

// expandUser replaces a leading ~ or $HOME with the path
// to the home directory of the current user
func expandUser(path string) string {
	// this is a simpler (and Linux/UNIX only) alternative to using os.UserHomeDir (which requires Go 1.12 or later)
	if strings.HasPrefix(path, "~") {
		path = strings.Replace(path, "~", os.Getenv("HOME"), 1)
	} else if strings.HasPrefix(path, "$HOME") {
		path = strings.Replace(path, "$HOME", os.Getenv("HOME"), 1)
	}
	return path
}

// hasAnyPrefixWord checks if the given line is prefixed with any one of the given words
func hasAnyPrefixWord(line string, wordList []string) bool {
	for _, word := range wordList {
		if strings.HasPrefix(line, word+" ") {
			return true
		}
	}
	return false
}

// hasS checks if the given string slice contains the given string
func hasS(sl []string, s string) bool {
	for _, e := range sl {
		if e == s {
			return true
		}
	}
	return false
}

// filterS returns all strings that makes the function f return true
func filterS(sl []string, f func(string) bool) []string {
	var results []string
	for _, e := range sl {
		if f(e) {
			results = append(results, e)
		}
	}
	return results
}

// nextGitRebaseKeywords takes the first word and increase it to the next git rebase keyword
func nextGitRebaseKeyword(line string) string {
	cycle1 := filterS(gitRebasePrefixes, func(s string) bool { return len(s) > 1 })
	cycle2 := filterS(gitRebasePrefixes, func(s string) bool { return len(s) == 1 })
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

// equalStringSlices checks if two given string slices are equal or not
// returns true if they are equal
func equalStringSlices(a, b []string) bool {
	lena := len(a)
	lenb := len(b)
	if lena != lenb {
		return false
	}
	for i := 0; i < lena; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
