package main

import (
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
