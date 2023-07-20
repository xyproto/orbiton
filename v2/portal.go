package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xyproto/env/v2"
)

var portalFilename = env.ExpandUser(filepath.Join(tempDir, env.Str("LOGNAME", "o")+"_portal.txt"))

// Portal is a filename and a line number, for pulling text from
type Portal struct {
	absFilename string
	lineNumber  LineNumber
}

// NewPortal returns a new portal to this filename and line number,
// but does not save the new portal. Use the Save() method for that.
func (e *Editor) NewPortal() (*Portal, error) {
	absFilename, err := e.AbsFilename()
	if err != nil {
		return nil, err
	}
	return &Portal{absFilename, e.LineNumber()}, nil
}

// SameFile checks if the portal exist in the same file as the editor is editing
func (p *Portal) SameFile(e *Editor) bool {
	absFilename, err := e.AbsFilename()
	if err != nil {
		return false
	}
	return absFilename == p.absFilename
}

// MoveDown is useful when using portals within the same file.
func (p *Portal) MoveDown() {
	// PopLine handles overflows.
	p.lineNumber++
}

// MoveUp is useful when using portals within the same file.
func (p *Portal) MoveUp() {
	// PopLine handles overflows.
	p.lineNumber--
}

// ClosePortal will clear the portal by removing the portal file
func (e *Editor) ClosePortal() error {
	e.sameFilePortal = nil
	return os.Remove(portalFilename)
}

// HasPortal checks if a portal is currently active
func HasPortal() bool {
	return exists(portalFilename)
}

// LoadPortal will load a filename + line number from the portal.txt file
func LoadPortal() (*Portal, error) {
	data, err := ReadFileNoStat(portalFilename)
	if err != nil {
		return nil, err
	}
	if !bytes.Contains(data, []byte{'\n'}) {
		return nil, errors.New(portalFilename + " does not have a newline, it's not a portal file")
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return nil, errors.New(portalFilename + " contains too few lines")
	}
	absFilename, err := filepath.Abs(lines[0])
	if err != nil {
		return nil, err
	}
	lineInt, err := strconv.Atoi(lines[1])
	if err != nil {
		return nil, err
	}
	lineNumber := LineNumber(lineInt)
	return &Portal{absFilename, lineNumber}, nil
}

// LineIndex returns the current line index that the portal points to
func (p *Portal) LineIndex() LineIndex {
	return p.lineNumber.LineIndex()
}

// LineNumber returns the current line number that the portal points to
func (p *Portal) LineNumber() LineNumber {
	return p.lineNumber
}

// Save will save the portal
func (p *Portal) Save() error {
	s := p.absFilename + "\n" + p.lineNumber.String() + "\n"
	// Anyone can read this file
	if err := os.WriteFile(portalFilename, []byte(s), 0o600); err != nil {
		return err
	}
	return os.Chmod(portalFilename, 0o666)
}

// String returns the current portal (filename + line number) as a colon separated string
func (p *Portal) String() string {
	return filepath.Base(p.absFilename) + ":" + p.lineNumber.String()
}

// NewLineInserted reacts when the editor inserts a new line in the same file,
// and moves the portal source one line down, if needed.
func (p *Portal) NewLineInserted(y LineIndex) {
	if y < p.LineIndex() {
		p.MoveDown()
	}
}

// PopLine removes (!) a line from the portal file, then removes that line
func (p *Portal) PopLine(e *Editor, removeLine bool) (string, error) {
	// popping a line from the same file is a special case
	if p == e.sameFilePortal {
		if removeLine {
			return "", errors.New("not implemented") // not implemented and currently not in use
		}
		// The line moving is done by the editor InsertAbove and InsertBelow functions
		return e.Line(p.LineIndex()), nil
	}
	data, err := ReadFileNoStat(p.absFilename)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	foundLine := ""
	found := false
	if removeLine {
		modifiedLines := make([]string, 0, len(lines)-1)
		for i, line := range lines {
			if LineIndex(i) == p.lineNumber.LineIndex() {
				foundLine = line
				found = true
			} else {
				modifiedLines = append(modifiedLines, line)
			}
		}
		if !found {
			return "", errors.New("Could not pop line " + p.String())
		}
		data = []byte(strings.Join(modifiedLines, "\n"))
		if err = os.WriteFile(p.absFilename, data, 0o600); err != nil {
			return "", err
		}
	} else {
		for i, line := range lines {
			if LineIndex(i) == p.lineNumber.LineIndex() {
				foundLine = line
				found = true
				break
			}
		}
		if !found {
			return "", errors.New("Could not pop line " + p.String())
		}
		// Now move the line number +1
		p.lineNumber++
		// And save the new portal
		if err := p.Save(); err != nil {
			return foundLine, err
		}
	}
	return foundLine, nil
}
