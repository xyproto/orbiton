package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	portalFilename = "~/.cache/o/portal.txt" // TODO: Use XDG_CACHE_HOME
)

// Portal is a filename and a line number, for pulling text from
type Portal struct {
	absFilename string
	lineNumber  LineNumber
}

// ClearPortal will clear the portal by removing the portal file
func ClearPortal() error {
	return os.Remove(expandUser(portalFilename))
}

// LoadPortal will load a filename + line number from the portal.txt file
func LoadPortal() (*Portal, error) {
	data, err := ioutil.ReadFile(expandUser(portalFilename))
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

// Save will save the portal
func (p *Portal) Save() error {
	s := p.absFilename + "\n" + p.lineNumber.String() + "\n"
	return ioutil.WriteFile(expandUser(portalFilename), []byte(s), 0600)
}

// String returns the current portal (filenane + linenumber) as a colon separated string
func (p *Portal) String() string {
	return filepath.Base(p.absFilename) + ":" + p.lineNumber.String()
}

// PopLine removes (!) a line from the portal file, then removes that line
func (p *Portal) PopLine() (string, error) {
	data, err := ioutil.ReadFile(p.absFilename)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	modifiedLines := make([]string, 0, len(lines)-1)
	foundLine := ""
	found := false
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
	if err = ioutil.WriteFile(p.absFilename, data, 0600); err != nil {
		return "", err
	}
	return foundLine, nil
}
