package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/xyproto/env"
)

// Read the HOME environment variable and default to /home/$LOGNAME if it isn't set
var homeDir = env.Str("HOME", "/home/"+os.Getenv("LOGNAME"))

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

// expandUser replaces a leading ~ or $HOME with the path
// to the home directory of the current user
func expandUser(path string) string {
	// this is a simpler (and Linux/UNIX only) alternative to using os.UserHomeDir (which requires Go 1.12 or later)
	if strings.HasPrefix(path, "~") {
		path = strings.Replace(path, "~", homeDir, 1)
	} else if strings.HasPrefix(path, "$HOME") {
		path = strings.Replace(path, "$HOME", homeDir, 1)
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

// isLower checks if all letters in a string are lowercase
// thanks: https://stackoverflow.com/a/59293875/131264
func isLower(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsLower(r) {
			return false
		}
	}
	return true
}

// Check if the given string only consists of the given rune,
// ignoring the other given runes.
func consistsOf(s string, e rune, ignore []rune) bool {
OUTER_LOOP:
	for _, r := range s {
		for _, x := range ignore {
			if r == x {
				continue OUTER_LOOP
			}
		}
		if r != e {
			//logf("CONSISTS OF: %s, %s, %s: FALSE\n", s, string(e), string(ignore))
			return false
		}
	}
	//logf("CONSISTS OF: %s, %s, %s: TRUE\n", s, string(e), string(ignore))
	return true
}

// aBinDirectory will check if the given filename is in one of these directories:
// /bin, /sbin, /usr/bin, /usr/sbin, /usr/local/bin, /usr/local/sbin, ~/.bin, ~/bin, ~/.local/bin
func aBinDirectory(filename string) bool {
	p, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return false
	}

	switch p {
	case "/bin", "/sbin", "/usr/bin", "/usr/sbin", "/usr/local/bin", "/usr/local/sbin":
		return true
	case filepath.Join(homeDir, ".bin"), filepath.Join(homeDir, "bin"), filepath.Join("local/bin"):
		return true
	}
	return false
}

// hexDigit checks if the given rune is 0-9, a-f, A-F or x
func hexDigit(r rune) bool {
	switch r {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'x', 'A', 'a', 'B', 'b', 'C', 'c', 'D', 'd', 'E', 'e', 'F', 'f':
		return true
	}
	return false
}

// firstLetterIsUpper checks if the first letter of the given string is uppercase
func firstLetterIsUpper(s string) bool {
	if len(s) == 0 {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsUpper(r)
}

// HasWords checks if a range of more than one letter is found
func HasWords(s string) bool {
	letterCount := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letterCount++
		} else {
			letterCount = 0
		}
		if letterCount > 1 {
			return true
		}
	}
	return false
}

func oneWordNoSpaces(s string) bool {
	letterCount := 0
	wordCount := 0
	for _, r := range s {
		if unicode.IsLetter(r) || r == '.' || r == '?' || r == '!' || r == '_' || r == ':' || r == '/' {
			letterCount++
		} else if r == ' ' {
			return false
		} else {
			if letterCount > 1 {
				wordCount++
				if wordCount > 1 {
					return false
				}
			}
			letterCount = 0
		}
	}
	return true
}

func oneField(s string) bool {
	return len(strings.Fields(s)) == 1
}

// logf, for quick "printf-style" debugging
func logf(format string, args ...interface{}) {
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
	f.WriteString(fmt.Sprintf(format, args...))
	f.Sync()
	f.Close()
}

// Silence the "logf is unused" message by staticcheck
var _ = logf
