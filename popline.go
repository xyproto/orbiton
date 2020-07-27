package main

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

// PopLineFrom can pop a line from the top of a file.
// This also modifies the file.
// permissions can be ie. 0600
func PopLineFrom(filename string, permissions os.FileMode) (string, error) {

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")

	foundLine := ""
	found := false

	if len(lines) == 0 || (len(lines) == 1 && len(strings.TrimSpace(lines[0])) == 0) {
		return "", errors.New("clipboard file is empty")
	}

	modifiedLines := make([]string, 0, len(lines)-1)
	for i, line := range lines {
		if LineIndex(i) == 0 {
			foundLine = line
			found = true
		} else {
			modifiedLines = append(modifiedLines, line)
		}
	}
	if !found {
		return "", errors.New("could not pop line from " + filename)
	}

	data = []byte(strings.Join(modifiedLines, "\n"))
	if err = ioutil.WriteFile(filename, data, permissions); err != nil {
		return foundLine, err
	}

	// Make the file available to other users if permissions are given
	if err := os.Chmod(filename, permissions); err != nil {
		return foundLine, err
	}

	return foundLine, nil

}

// PushLineTo can push a line to the bottom of a file.
// permissions can be ie. 0600
func PushLineTo(filename, line string, permissions os.FileMode) error {

	if !exists(filename) {
		if _, err := os.Create(filename); err != nil {
			return err
		}
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")

	// Append line to the top of the lines
	lines = append(lines, line)

	// Write the lines to file
	data = []byte(strings.Join(lines, "\n"))
	if err = ioutil.WriteFile(filename, data, permissions); err != nil {
		return err
	}

	// Make the file available to other users if permissions are given
	if err := os.Chmod(filename, permissions); err != nil {
		return err
	}

	return nil
}
