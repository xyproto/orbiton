package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/xyproto/mode"
)

// manSample is shaped like real "man" output: every bold character is doubled
// around a backspace, underlines are _\bX, plus UTF-8 punctuation and bullets
const manSample = "LS(1)\n" +
	"N\bNA\bAM\bME\bE\n" +
	"     l\bls\bs — list directory contents\n" +
	"     _\bi_\bt_\be_\bm\n" +
	"     •   not defined in IEEE Std 1003.1-2008 (“POSIX.1”)\n" +
	"D\bDE\bES\bSC\bCR\bRI\bIP\bPT\bTI\bIO\bON\bN\n" +
	"     T\bTh\bhe\be following options are available:\n"

// The backspaces make man output look like binary data. Loading it one rune per
// byte turns every UTF-8 sequence into mojibake, so man pages must be decoded
// as text.
func TestManPageIsNotLoadedAsBinary(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.ManPage
	e.LoadBytes([]byte(manSample))
	if e.binaryFile {
		t.Error("a man page should not be treated as a binary file")
	}
	s := e.String()
	if strings.Contains(s, "â") {
		t.Errorf("UTF-8 was decoded one rune per byte: %q", s)
	}
	for _, want := range []string{"—", "•", "“", "”"} {
		if !strings.Contains(s, want) {
			t.Errorf("%q is missing from the loaded man page", want)
		}
	}
}

func TestStripManPageEscapes(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.ManPage
	e.LoadBytes([]byte(manSample))
	e.stripManPageEscapes()
	s := e.String()
	if strings.Contains(s, "\b") {
		t.Errorf("backspaces were left in the buffer: %q", s)
	}
	for _, want := range []string{"ls — list directory contents", "item", "•   not defined", "(“POSIX.1”)"} {
		if !strings.Contains(s, want) {
			t.Errorf("%q is missing from %q", want, s)
		}
	}
}

var colorAndRune = regexp.MustCompile(`\x1b\[([0-9;]*)m([^\x1b])`)

// colorRun is a run of characters that were all drawn with the same color
type colorRun struct {
	color string
	text  string
}

// colorRuns groups the highlighted text by color
func colorRuns(highlighted string) []colorRun {
	var runs []colorRun
	for _, m := range colorAndRune.FindAllStringSubmatch(highlighted, -1) {
		if n := len(runs); n > 0 && runs[n-1].color == m[1] {
			runs[n-1].text += m[2]
			continue
		}
		runs = append(runs, colorRun{color: m[1], text: m[2]})
	}
	return runs
}

func highlightManLine(t *testing.T, line string) []colorRun {
	t.Helper()
	e := NewSimpleEditor(80)
	e.mode = mode.ManPage
	return colorRuns(e.manPageHighlight(line, false, false))
}

// colorOf returns the color of the first run that is exactly text
func colorOf(runs []colorRun, text string) string {
	for _, run := range runs {
		if run.text == text {
			return run.color
		}
	}
	return ""
}

// A "[-abcABC]" cluster is a list of flag letters, so the letters in it should
// all look the same instead of the uppercase ones standing out
func TestSynopsisFlagClusterIsOneColor(t *testing.T) {
	const line = "     ls [-@ABCFGHILOPRSTUWabcdefghiklmnopqrstuvwxy1%,] [file ...]"
	runs := highlightManLine(t, line)
	if len(runs) != 1 {
		t.Errorf("the whole line should be one color, got %d runs:", len(runs))
		for _, run := range runs {
			t.Errorf("  %s %q", run.color, run.text)
		}
	}
}

// Uppercase words outside a flag cluster must still be highlighted
func TestUppercaseWordsAreStillHighlighted(t *testing.T) {
	runs := highlightManLine(t, "     conforms to the POSIX standard")
	upper := colorOf(runs, "POSIX")
	if upper == "" {
		t.Fatalf("POSIX was not drawn as one run: %v", runs)
	}
	if plain := colorOf(runs, "     conforms to the "); upper == plain {
		t.Errorf("POSIX is %q, the same as ordinary text", upper)
	}
}

// "[FILE]" is not a flag cluster, so it is left as it was
func TestPlainBracketsAreNotFlagClusters(t *testing.T) {
	runs := highlightManLine(t, "     cat [FILE]")
	if colorOf(runs, "FILE") == "" {
		t.Errorf("FILE in plain brackets should still be one highlighted run: %v", runs)
	}
}

// An email address in angle brackets gets a color of its own, with the brackets
// and the "@" set apart from the parts around them
func TestEmailAddressIsHighlighted(t *testing.T) {
	runs := highlightManLine(t, "     Report bugs to <htop@groups.io>")
	prose := colorOf(runs, "     Report bugs to ")
	for _, part := range []string{"htop", "groups.io"} {
		if c := colorOf(runs, part); c == "" || c == prose {
			t.Errorf("%q is %q, want a color of its own, not the prose color %q", part, c, prose)
		}
	}
	bracket := colorOf(runs, "<")
	for _, part := range []string{"@", ">"} {
		if c := colorOf(runs, part); c != bracket {
			t.Errorf("%q is %q, want %q like the opening bracket", part, c, bracket)
		}
	}
}

// wastedColors counts color escapes that no visible character follows
func wastedColors(highlighted string) int {
	n := 0
	for i, part := range strings.Split(highlighted, "\x1b") {
		if i == 0 {
			continue
		}
		end := strings.IndexByte(part, 'm')
		if end < 0 || end+1 != len(part) || part[:end+1] == "[0m" {
			continue
		}
		n++
	}
	return n
}

// Every color that is emitted should apply to at least one character
func TestNoWastedColors(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.ManPage
	for _, line := range []string{
		"     ls [-@ABC] file",
		"     Contact user@example.com for help",
		"     mail root@localhost now",
		"@ at the line start",
		"     ends with @",
		"     user@host:1234 and <TAB>",
		"     0x1F@2A and 123@456 here",
		"     plain text with UPPER words",
	} {
		if n := wastedColors(e.manPageHighlight(line, false, false)); n != 0 {
			t.Errorf("%q emits %d color escapes that nothing follows", line, n)
		}
	}
}
