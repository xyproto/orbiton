package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const locationHistoryFilename = "~/.config/o/locations.txt"

// expandUser replaces a leading ~ or $HOME with the path
// to the home directory of the current user
func expandUser(path string) string {
	// this is a simpler alternative to using os.UserHomeDir (which requires Go 1.12 or later)
	if strings.HasPrefix(path, "~") {
		path = strings.Replace(path, "~", os.Getenv("HOME"), 1)
	} else if strings.HasPrefix(path, "$HOME") {
		path = strings.Replace(path, "$HOME", os.Getenv("HOME"), 1)
	}
	return path
}

// LoadLocationHistory will attempt to load the per-absolute-filename recording of which line is active
func LoadLocationHistory(configFile string) map[string]int {
	locationHistory := make(map[string]int)

	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		// Could not read file, return an empty map
		return locationHistory
	}
	// The format of the file is, per line:
	// "filename":location
	for _, filenameLocation := range strings.Split(string(contents), "\n") {
		if !strings.Contains(filenameLocation, ":") {
			continue
		}
		fields := strings.SplitN(filenameLocation, ":", 2)

		// Retrieve an unquoted filename in the filename variable
		quotedFilename := strings.TrimSpace(fields[0])
		filename := quotedFilename
		if strings.HasPrefix(quotedFilename, "\"") && strings.HasSuffix(quotedFilename, "\"") {
			filename = quotedFilename[1 : len(quotedFilename)-1]
		}
		if filename == "" {
			continue
		}

		// Retrieve the line number
		lineNumberString := strings.TrimSpace(fields[1])
		lineNumber, err := strconv.Atoi(lineNumberString)
		if err != nil {
			// Could not convert to a number
			continue
		}
		locationHistory[filename] = lineNumber
	}

	// Return the location history map. It could be empty, which is fine.
	return locationHistory
}

// SaveLocationHistory will attempt to save the per-absolute-filename recording of which line is active
func SaveLocationHistory(locationHistory map[string]int, configFile string) {
	folderPath := filepath.Dir(configFile)

	// First create the folder, if needed, in a best effort attempt
	os.MkdirAll(folderPath, os.ModePerm)

	var sb strings.Builder
	for k, v := range locationHistory {
		sb.WriteString(fmt.Sprintf("\"%s\": %d\n", k, v))
	}
	// Ignore errors, this is a best effort attempt
	_ = ioutil.WriteFile(configFile, []byte(sb.String()), 0644)
}

// SaveLocation takes a filename (which includes the absolute path) and a map which contains
// an overview of which files were at which line location.
func (e *Editor) SaveLocation(absFilename string, locationHistory map[string]int) {
	// Save the current line location
	locationHistory[absFilename] = e.LineNumber()
	// Save the location history (best effort, ignore errors)
	SaveLocationHistory(locationHistory, expandUser(locationHistoryFilename))
}
