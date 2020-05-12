package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	locationHistoryFilename      = "~/.cache/o/locations.txt" // TODO: Use XDG_CACHE_HOME
	vimLocationHistoryFilename   = "~/.viminfo"
	emacsLocationHistoryFilename = "~/.emacs.d/places"
	nvimLocationHistoryFilename  = "~/.local/share/nvim/shada/main.shada" // TODO: Use XDG_DATA_HOME
	maxLocationHistoryEntries    = 1024
)

var (
	locationHistory map[string]LineNumber // remember where we were in each absolute filename
)

// LoadLocationHistory will attempt to load the per-absolute-filename recording of which line is active.
// The returned map can be empty.
func LoadLocationHistory(configFile string) (map[string]LineNumber, error) {
	locationHistory := make(map[string]LineNumber)

	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		// Could not read file, return an empty map and an error
		return locationHistory, err
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
		locationHistory[filename] = LineNumber(lineNumber)
	}

	// Return the location history map. It could be empty, which is fine.
	return locationHistory, nil
}

// LoadVimLocationHistory will attempt to load the history of where the cursor should be when opening a file from ~/.viminfo
// The returned map can be empty. The filenames have absolute paths.
func LoadVimLocationHistory(vimInfoFilename string) map[string]LineNumber {
	locationHistory := make(map[string]LineNumber)
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
			locationHistory[absFilename] = LineNumber(lineNumber)
		}
	}
	return locationHistory
}

// FindInVimLocationHistory will try to find the given filename in the ViM .viminfo file
func FindInVimLocationHistory(vimInfoFilename, searchFilename string) (LineNumber, error) {
	// Attempt to read the ViM location history (that may or may not exist)
	data, err := ioutil.ReadFile(vimInfoFilename)
	if err != nil {
		return LineNumber(-1), err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "-'") {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			lineNumberString := fields[1]
			filename := fields[3]
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
			if absFilename == searchFilename {
				return LineNumber(lineNumber), nil
			}
		}
	}
	return LineNumber(-1), errors.New("filename not found in vim location history: " + searchFilename)
}

// FindInNvimLocationHistory will try to find the given filename in the NeoVim location history file
func FindInNvimLocationHistory(nvimLocationFilename, searchFilename string) (LineNumber, error) {
	nol := LineNumber(-1) // no line number

	data, err := ioutil.ReadFile(nvimLocationFilename) // typically main.shada, a MsgPack file
	if err != nil {
		return nol, err
	}

	pos := bytes.Index(data, []byte(searchFilename))

	if pos < 0 {
		return nol, errors.New("filename not found in nvim location history: " + searchFilename)
	}

	if pos < 2 {
		// this should never happen
		return nol, errors.New("too early match in file")
	}

	dataType := data[pos-2]
	if dataType != 0xc4 {
		// this should never happen
		return nol, errors.New("not a binary data type")
	}

	stringLength := int(data[pos-1])
	if len(searchFilename) != stringLength {
		// this should never happen
		return nol, errors.New("mismatching string length")
	}

	maxi := len(data) - 1
	nextNumberIsTheLineNumber := false

	pp := pos - 2

	// Search 500 bytes from here, at a maximum
	for i := 0; i < 500; i++ {
		if (pp + i) >= maxi {
			return nol, errors.New("corresponding line not found for " + searchFilename)
		}
		b := data[pp+i] // The current byte
		//nb := data[pp+i+1] // The next byte

		//fmt.Printf("--- byte pos %d [value %x]---\n", i, b)

		switch {
		case b <= 0x7f: //&& b >= 0
			//fmt.Printf("%d (positive fixint)\n", b)
			if nextNumberIsTheLineNumber {
				//fmt.Println("FOUND THE LINE NUMBER FOR " + searchFilename + "!")
				//fmt.Printf("=== %d ===\n", int(b))
				return LineNumber(int(b)), nil
			}
		case b >= 0x80 && b <= 0x8f:
			size := uint(b - 128) // - b10000000
			size--
			size *= 2
			bd := data[pp+i+1 : pp+i+1+int(size)]
			i += int(size)
			_ = bd
			//fmt.Printf("%s (fixmap, size %d)\n", bd, size)
		case b >= 0x90 && b <= 0x9f:
			size := uint(b - 144) // - b10010000
			size--
			bd := data[pp+i+1 : pp+i+1+int(size)]
			i += int(size)
			_ = bd
			//fmt.Printf("%s (fixarray, size %d)\n", bd, size)
		case b >= 0xa0 && b <= 0xbf:
			size := uint(b - 160) // - 101xxxxx
			bd := data[pp+i+1 : pp+i+1+int(size)]
			i += int(size)
			//fmt.Printf("%s (fixstr, size %d)\n", string(bd), size)
			if string(bd) == "l" {
				// Found a NeoVim line number string "l", specifying that the next
				// element is a number.
				nextNumberIsTheLineNumber = true
			}
		case b == 0xc0:
			//fmt.Println("nil")
		case b == 0xc1:
			//fmt.Println("<unused>")
		case b == 0xc2:
			//fmt.Println("false")
		case b == 0xc3:
			//fmt.Println("true")
		case b == 0xc4:
			i++
			size := uint(data[pp+i])
			bd := data[pp+i+1 : pp+i+1+int(size)]
			i += int(size)
			_ = bd
			//fmt.Printf("%s (bin 8, size %d)\n", string(bd), size)
		case b == 0xc5:
			//fmt.Println("bin 16")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: bin 16")
		case b == 0xc6:
			//fmt.Println("bin 32")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: bin 32")
		case b == 0xc7:
			//fmt.Println("ext 8")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: ext 8")
		case b == 0xc8:
			//fmt.Println("ext 16")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: ext 16")
		case b == 0xc9:
			//fmt.Println("ext 32")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: ext 32")
		case b == 0xca:
			//fmt.Println("float 32")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: float 32")
		case b == 0xcb:
			//fmt.Println("float 64")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: float 64")
		case b == 0xcc:
			i++
			d0 := data[pp+i]
			l := d0
			//fmt.Printf("%d (uint 8)\n", l)
			if nextNumberIsTheLineNumber {
				//fmt.Println("FOUND THE LINE NUMBER FOR " + searchFilename + "!")
				//fmt.Printf("=== %d ===\n", l)
				return LineNumber(l), nil
			}
		case b == 0xcd:
			i++
			d0 := data[pp+i]
			i++
			d1 := data[pp+i]
			l := uint16(d0)<<8 + uint16(d1)
			//fmt.Printf("%d (uint 16)\n", l)
			if nextNumberIsTheLineNumber {
				//fmt.Println("FOUND THE LINE NUMBER FOR " + searchFilename + "!")
				//fmt.Printf("=== %d ===\n", l)
				return LineNumber(l), nil
			}
		case b == 0xce:
			i++
			d0 := data[pp+i]
			i++
			d1 := data[pp+i]
			i++
			d2 := data[pp+i]
			i++
			d3 := data[pp+i]
			l := uint32(d0)<<24 + uint32(d1)<<16 + uint32(d2)<<8 + uint32(d3)
			//fmt.Printf("%d (uint 32)\n", l)
			if nextNumberIsTheLineNumber {
				//fmt.Println("FOUND THE LINE NUMBER FOR " + searchFilename + "!")
				//fmt.Printf("=== %d ===\n", l)
				return LineNumber(l), nil
			}
		case b == 0xcf:
			i++
			d0 := data[pp+i]
			i++
			d1 := data[pp+i]
			i++
			d2 := data[pp+i]
			i++
			d3 := data[pp+i]
			i++
			d4 := data[pp+i]
			i++
			d5 := data[pp+i]
			i++
			d6 := data[pp+i]
			i++
			d7 := data[pp+i]
			l := uint64(d0)<<56 + uint64(d1)<<48 + uint64(d2)<<40 + uint64(d3)<<32 + uint64(d4)<<24 + uint64(d5)<<16 + uint64(d6)<<8 + uint64(d7)
			//fmt.Printf("%d (uint 64)\n", l)
			if nextNumberIsTheLineNumber {
				//fmt.Println("FOUND THE LINE NUMBER FOR " + searchFilename + "!")
				//fmt.Printf("=== %d ===\n", l)
				return LineNumber(l), nil
			}
		case b == 0xd0:
			//fmt.Println("int 8")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: int 8")
		case b == 0xd1:
			//fmt.Println("int 16")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: int 16")
		case b == 0xd2:
			//fmt.Println("int 32")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: int 32")
		case b == 0xd3:
			//fmt.Println("int 64")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: int 64")
		case b == 0xd4:
			//fmt.Println("fixext 1")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: fixext 1")
		case b == 0xd5:
			//fmt.Println("fixext 2")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: fixext 2")
		case b == 0xd6:
			//fmt.Println("fixext 4")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: fixext 4")
		case b == 0xd7:
			//fmt.Println("fixext 8")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: fixext 8")
		case b == 0xd8:
			//fmt.Println("fixext 16")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: fixext 16")
		case b == 0xd9:
			//fmt.Println("str 8")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: str 8")
		case b == 0xda:
			//fmt.Println("str 16")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: str 16")
		case b == 0xdb:
			//fmt.Println("str 32")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: str 32")
		case b == 0xdc:
			//fmt.Println("array 16")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: array 16")
		case b == 0xdd:
			//fmt.Println("array 32")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: array 32")
		case b == 0xde:
			//fmt.Println("map 16")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: map 16")
		case b == 0xdf:
			//fmt.Println("map 32")
			//panic("unimplemented")
			return nol, errors.New("unimplemented msgpack field: map 32")
		case b >= 0xe0 && b <= 0xff:
			n := -(int(b) - 224) // - 111xxxxx
			_ = n
			//fmt.Printf("%d (negative fixint)\n", n)
		default:
			return nol, fmt.Errorf("unrecognized msgpack field: %x", b)
		}
	}
	return nol, errors.New("could not find line number for " + searchFilename)
}

// LoadEmacsLocationHistory will attempt to load the history of where the cursor should be when opening a file from ~/.emacs.d/places.
// The returned map can be empty. The filenames have absolute paths.
// The values in the map are NOT line numbers but character positions.
func LoadEmacsLocationHistory(emacsPlacesFilename string) map[string]CharacterPosition {
	locationHistory := make(map[string]CharacterPosition)
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
		locationHistory[absFilename] = CharacterPosition(charNumber)
	}
	return locationHistory
}

// SaveLocationHistory will attempt to save the per-absolute-filename recording of which line is active
func SaveLocationHistory(locationHistory map[string]LineNumber, configFile string) error {
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
func (e *Editor) SaveLocation(absFilename string, locationHistory map[string]LineNumber) error {
	if len(locationHistory) > maxLocationHistoryEntries {
		// Cull the history
		locationHistory = make(map[string]LineNumber, 1)
	}
	// Save the current line location
	locationHistory[absFilename] = e.LineNumber()
	// Save the location history and return the error, if any
	return SaveLocationHistory(locationHistory, expandUser(locationHistoryFilename))
}
