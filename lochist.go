package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	locationHistoryFilename      = "~/.cache/o/locations.txt"
	vimLocationHistoryFilename   = "~/.viminfo"
	emacsLocationHistoryFilename = "~/.emacs.d/places"
	maxLocationHistoryEntries    = 1024
)

// LoadLocationHistory will attempt to load the per-absolute-filename recording of which line is active.
// The returned map can be empty.
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

// LoadVimLocationHistory will attempt to load the history of where the cursor should be when opening a file from ~/.viminfo
// The returned map can be empty. The filenames have absolute paths.
func LoadVimLocationHistory(vimInfoFilename string) map[string]int {
	locationHistory := make(map[string]int)
	// Attempt to read the ViM location history (that may or may not exist)
	data, err := ioutil.ReadFile(vimInfoFilename)
	if err != nil {
		return locationHistory
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "-'") {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			lineNumberString := fields[1]
			//colNumberString := fields[2]
			filename := fields[3]
			// Skip if the filename already exists in the location history, since .viminfo
			// may have duplication locations and lists the newest first.
			if _, alreadyExists := locationHistory[filename]; alreadyExists {
				continue
			}
			//fmt.Println("LINE NUMBER", lineNumberString, "FILENAME", filename)
			lineNumber, err := strconv.Atoi(lineNumberString)
			if err != nil {
				// Not a line number after all
				continue
			}
			absFilename, err := filepath.Abs(filename)
			if err != nil {
				// Could not get the absolute path
				continue
			}
			locationHistory[absFilename] = lineNumber
		}
	}
	return locationHistory
}

// LoadEmacsLocationHistory will attempt to load the history of where the cursor should be when opening a file from ~/.emacs.d/places.
// The returned map can be empty. The filenames have absolute paths.
// The values in the map are NOT line numbers but character positions.
func LoadEmacsLocationHistory(emacsPlacesFilename string) map[string]int {
	locationHistory := make(map[string]int)
	// Attempt to read the Emacs location history (that may or may not exist)
	data, err := ioutil.ReadFile(emacsPlacesFilename)
	if err != nil {
		return locationHistory
	}
	for _, line := range strings.Split(string(data), "\n") {
		// Looking for lines with filenames with ""
		fields := strings.SplitN(line, "\"", 3)
		if len(fields) != 3 {
			continue
		}
		filename := fields[1]
		locationAndMore := fields[2]
		// Strip trailing parenthesis
		for strings.HasSuffix(locationAndMore, ")") {
			locationAndMore = locationAndMore[:len(locationAndMore)-1]
		}
		fields = strings.Fields(locationAndMore)
		if len(fields) == 0 {
			continue
		}
		lastField := fields[len(fields)-1]
		charNumber, err := strconv.Atoi(lastField)
		if err != nil {
			// Not a character number
			continue
		}
		absFilename, err := filepath.Abs(filename)
		if err != nil {
			// Could not get absolute path
			continue
		}
		locationHistory[absFilename] = charNumber
	}
	return locationHistory
}

// SaveLocationHistory will attempt to save the per-absolute-filename recording of which line is active
func SaveLocationHistory(locationHistory map[string]int, configFile string) error {
	folderPath := filepath.Dir(configFile)

	// First create the folder, if needed, in a best effort attempt
	os.MkdirAll(folderPath, os.ModePerm)

	var sb strings.Builder
	for k, v := range locationHistory {
		sb.WriteString(fmt.Sprintf("\"%s\": %d\n", k, v))
	}
	// Write the location history and return the error, if any
	return ioutil.WriteFile(configFile, []byte(sb.String()), 0644)
}

// SaveLocation takes a filename (which includes the absolute path) and a map which contains
// an overview of which files were at which line location.
func (e *Editor) SaveLocation(absFilename string, locationHistory map[string]int) error {
	if len(locationHistory) > maxLocationHistoryEntries {
		// Cull the history
		locationHistory = make(map[string]int, 1)
	}
	// Save the current line location
	locationHistory[absFilename] = e.LineNumber()
	// Save the location history and return the error, if any
	return SaveLocationHistory(locationHistory, expandUser(locationHistoryFilename))
}
