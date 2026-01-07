package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxLocationHistoryEntries = 1024

var locationHistory LocationHistory // per absolute filename, for jumping to the last used line when opening a file

// LineNumberAndTimestamp contains both a LineNumber and a time.Time
type LineNumberAndTimestamp struct {
	Timestamp  time.Time
	LineNumber LineNumber
}

// LocationHistory stores the absolute path to a filename, a line number and a timestamp (for trimming the location history)
type LocationHistory map[string]LineNumberAndTimestamp

// Has checks if the location history has the given absolute path
func (locationHistory LocationHistory) Has(path string) bool {
	_, found := locationHistory[path]
	return found
}

// Get takes an absolute path and returns the line number and true if found
func (locationHistory LocationHistory) Get(path string) (LineNumber, bool) {
	if lnat, found := locationHistory[path]; found {
		return lnat.LineNumber, true
	}
	return 0, false
}

// Set sets a new line number for the given absolute path, and also records the current time
func (locationHistory LocationHistory) Set(path string, ln LineNumber) {
	var lnat LineNumberAndTimestamp
	lnat.Timestamp = time.Now()
	lnat.LineNumber = ln
	locationHistory[path] = lnat
}

// SetWithTimestamp sets a new line number for the given absolute path, and also records the current time
func (locationHistory LocationHistory) SetWithTimestamp(path string, ln LineNumber, timestamp int64) {
	var lnat LineNumberAndTimestamp
	lnat.Timestamp = time.Unix(timestamp, 0)
	lnat.LineNumber = ln
	locationHistory[path] = lnat
}

// Save will attempt to save the per-absolute-filename recording of which line is active
func (locationHistory LocationHistory) Save(path string) error {
	if noWriteToCache {
		return nil
	}
	// First create the folder, if needed, in a best effort attempt
	folderPath := filepath.Dir(path)
	_ = os.MkdirAll(folderPath, 0o755) // try to (re)create the directory, but ignore errors
	var sb strings.Builder
	for k, lineNumberAndTimestamp := range locationHistory {
		lineNumber := lineNumberAndTimestamp.LineNumber
		timeStamp := lineNumberAndTimestamp.Timestamp
		sb.WriteString(fmt.Sprintf("%d:%d:%s\n", timeStamp.Unix(), lineNumber, k))
	}
	// Write the location history and return the error, if any.
	// The permissions are a bit stricter for this one.
	return os.WriteFile(path, []byte(sb.String()), 0o600)
}

// Len returns the current location history length
func (locationHistory LocationHistory) Len() int {
	return len(locationHistory)
}

// LoadLocationHistory will attempt to load the per-absolute-filename recording of which line is active.
// The returned map can be empty.
func LoadLocationHistory(configFile string) (LocationHistory, error) {
	locationHistory := make(LocationHistory)

	contents, err := os.ReadFile(configFile)
	if err != nil {
		// Could not read file, return an empty map and an error
		return locationHistory, err
	}
	// The format of the file is, per line:
	// "filename":location
	for _, filenameLocation := range strings.Split(string(contents), "\n") {
		fields := strings.Split(filenameLocation, ":")

		if len(fields) == 2 {

			// Retrieve an unquoted filename in the filename variable
			quotedFilename := strings.TrimSpace(fields[0])
			absFilename := quotedFilename
			if strings.HasPrefix(quotedFilename, "\"") && strings.HasSuffix(quotedFilename, "\"") {
				absFilename = quotedFilename[1 : len(quotedFilename)-1]
			}
			if absFilename == "" || !ShouldKeep(absFilename) {
				continue
			}

			lineNumberAndMaybeTimestamp := fields[1]

			// Retrieve the line number
			lineNumberString := strings.TrimSpace(lineNumberAndMaybeTimestamp)
			lineNumber, err := strconv.Atoi(lineNumberString)
			if err != nil {
				// Could not convert to a number
				continue
			}
			locationHistory.Set(absFilename, LineNumber(lineNumber))

		} else if len(fields) == 3 {

			// Retrieve the line number and UNIX timestamp
			timeStampString := fields[0]
			lineNumberString := fields[1]
			absFilename := fields[2]

			if !ShouldKeep(absFilename) {
				// Typically files in /tmp
				continue
			}

			lineNumber, err := strconv.Atoi(lineNumberString)
			if err != nil {
				// Could not convert to a number
				continue
			}

			timestamp, err := strconv.ParseInt(timeStampString, 10, 64)
			if err != nil {
				// Could not convert to a number
				continue
			}

			locationHistory.SetWithTimestamp(absFilename, LineNumber(lineNumber), timestamp)
		}

	}

	// Return the location history map. It could be empty, which is fine.
	return locationHistory, nil
}

// LoadVimLocationHistory will attempt to load the history of where the cursor should be when opening a file from ~/.viminfo
// The returned map can be empty. The filenames have absolute paths.
func LoadVimLocationHistory(vimInfoFilename string) LocationHistory {
	locationHistory := make(LocationHistory)
	// Attempt to read the ViM location history (that may or may not exist)
	data, err := os.ReadFile(vimInfoFilename)
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
			// colNumberString := fields[2]
			filename := fields[3]
			// Skip if the filename already exists in the location history, since .viminfo
			// may have duplication locations and lists the newest first.
			if locationHistory.Has(filename) {
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
			absFilename = filepath.Clean(absFilename)
			locationHistory.Set(absFilename, LineNumber(lineNumber))
		}
	}
	return locationHistory
}

// FindInVimLocationHistory will try to find the given filename in the ViM .viminfo file
func FindInVimLocationHistory(vimInfoFilename, searchFilename string) (LineNumber, error) {
	// Attempt to read the ViM location history (that may or may not exist)
	data, err := os.ReadFile(vimInfoFilename)
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
			absFilename = filepath.Clean(absFilename)
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

	data, err := os.ReadFile(nvimLocationFilename) // typically main.shada, a MsgPack file
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

	maxi := len(data) - 1
	nextNumberIsTheLineNumber := false

	pp := pos - 2

	// Search 512 bytes from here, at a maximum
	for i := 0; i < 512; i++ {
		if (pp + i) >= maxi {
			return nol, errors.New("corresponding line not found for " + searchFilename)
		}
		b := data[pp+i] // The current byte

		// fmt.Printf("--- byte pos %d [value %x]---\n", i, b)

		switch {
		case b <= 0x7f: // fixint
			// fmt.Printf("%d (positive fixint)\n", b)
			if nextNumberIsTheLineNumber {
				return LineNumber(int(b)), nil
			}
		case b >= 0x80 && b <= 0x8f: // fixmap
			size := uint(b - 128) // - b10000000
			size--
			size *= 2
			bd := data[pp+i+1 : pp+i+1+int(size)]
			i += int(size)
			_ = bd
			// fmt.Printf("%s (fixmap, size %d)\n", bd, size)
		case b >= 0x90 && b <= 0x9f: // fixarray
			size := uint(b - 144) // - b10010000
			size--
			bd := data[pp+i+1 : pp+i+1+int(size)]
			i += int(size)
			_ = bd
			// fmt.Printf("%s (fixarray, size %d)\n", bd, size)
		case b >= 0xa0 && b <= 0xbf: // fixstr
			size := uint(b - 160) // - 101xxxxx
			bd := data[pp+i+1 : pp+i+1+int(size)]
			i += int(size)
			// fmt.Printf("%s (fixstr, size %d)\n", string(bd), size)
			if string(bd) == "l" {
				// Found a NeoVim line number string "l", specifying that the next
				// element is a number.
				nextNumberIsTheLineNumber = true
			}
		case b == 0xc0: // nil
			// fmt.Println("nil")
		case b == 0xc1: // unused
			// fmt.Println("<unused>")
		case b == 0xc2: // false
			// fmt.Println("false")
		case b == 0xc3: // true
			// fmt.Println("true")
		case b == 0xc4: // bin 8
			i++
			size := uint(data[pp+i])
			bd := data[pp+i+1 : pp+i+1+int(size)]
			i += int(size)
			_ = bd
			// fmt.Printf("%s (bin 8, size %d)\n", string(bd), size)
		case b == 0xc5:
			// fmt.Println("bin 16")
			return nol, errors.New("unimplemented msgpack field: bin 16")
		case b == 0xc6:
			// fmt.Println("bin 32")
			return nol, errors.New("unimplemented msgpack field: bin 32")
		case b == 0xc7:
			// fmt.Println("ext 8")
			return nol, errors.New("unimplemented msgpack field: ext 8")
		case b == 0xc8:
			// fmt.Println("ext 16")
			return nol, errors.New("unimplemented msgpack field: ext 16")
		case b == 0xc9:
			// fmt.Println("ext 32")
			return nol, errors.New("unimplemented msgpack field: ext 32")
		case b == 0xca:
			// fmt.Println("float 32")
			return nol, errors.New("unimplemented msgpack field: float 32")
		case b == 0xcb:
			// fmt.Println("float 64")
			return nol, errors.New("unimplemented msgpack field: float 64")
		case b == 0xcc: // uint 8
			i++
			d0 := data[pp+i]
			l := d0
			// fmt.Printf("%d (uint 8)\n", l)
			if nextNumberIsTheLineNumber {
				return LineNumber(l), nil
			}
		case b == 0xcd: // uint 16
			i++
			d0 := data[pp+i]
			i++
			d1 := data[pp+i]
			l := uint16(d0)<<8 + uint16(d1)
			// fmt.Printf("%d (uint 16)\n", l)
			if nextNumberIsTheLineNumber {
				return LineNumber(l), nil
			}
		case b == 0xce: // uint 32
			i++
			d0 := data[pp+i]
			i++
			d1 := data[pp+i]
			i++
			d2 := data[pp+i]
			i++
			d3 := data[pp+i]
			l := uint32(d0)<<24 + uint32(d1)<<16 + uint32(d2)<<8 + uint32(d3)
			// fmt.Printf("%d (uint 32)\n", l)
			if nextNumberIsTheLineNumber {
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
			// fmt.Printf("%d (uint 64)\n", l)
			if nextNumberIsTheLineNumber {
				return LineNumber(l), nil
			}
		case b == 0xd0:
			// fmt.Println("int 8")
			return nol, errors.New("unimplemented msgpack field: int 8")
		case b == 0xd1:
			// fmt.Println("int 16")
			return nol, errors.New("unimplemented msgpack field: int 16")
		case b == 0xd2:
			// fmt.Println("int 32")
			return nol, errors.New("unimplemented msgpack field: int 32")
		case b == 0xd3:
			// fmt.Println("int 64")
			return nol, errors.New("unimplemented msgpack field: int 64")
		case b == 0xd4:
			// fmt.Println("fixext 1")
			return nol, errors.New("unimplemented msgpack field: fixext 1")
		case b == 0xd5:
			// fmt.Println("fixext 2")
			return nol, errors.New("unimplemented msgpack field: fixext 2")
		case b == 0xd6:
			// fmt.Println("fixext 4")
			return nol, errors.New("unimplemented msgpack field: fixext 4")
		case b == 0xd7:
			// fmt.Println("fixext 8")
			return nol, errors.New("unimplemented msgpack field: fixext 8")
		case b == 0xd8:
			// fmt.Println("fixext 16")
			return nol, errors.New("unimplemented msgpack field: fixext 16")
		case b == 0xd9:
			// fmt.Println("str 8")
			return nol, errors.New("unimplemented msgpack field: str 8")
		case b == 0xda:
			// fmt.Println("str 16")
			return nol, errors.New("unimplemented msgpack field: str 16")
		case b == 0xdb:
			// fmt.Println("str 32")
			return nol, errors.New("unimplemented msgpack field: str 32")
		case b == 0xdc:
			// fmt.Println("array 16")
			return nol, errors.New("unimplemented msgpack field: array 16")
		case b == 0xdd:
			// fmt.Println("array 32")
			return nol, errors.New("unimplemented msgpack field: array 32")
		case b == 0xde:
			// fmt.Println("map 16")
			return nol, errors.New("unimplemented msgpack field: map 16")
		case b == 0xdf:
			// fmt.Println("map 32")
			return nol, errors.New("unimplemented msgpack field: map 32")
		case b >= 0xe0: // >= 0xff is implied
			n := -(int(b) - 224) // - 111xxxxx
			_ = n
			// fmt.Printf("%d (negative fixint)\n", n)
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
	locationCharHistory := make(map[string]CharacterPosition)
	// Attempt to read the Emacs location history (that may or may not exist)
	data, err := os.ReadFile(emacsPlacesFilename)
	if err != nil {
		return locationCharHistory
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
		absFilename = filepath.Clean(absFilename)
		locationCharHistory[absFilename] = CharacterPosition(charNumber)
	}
	return locationCharHistory
}

// ShouldKeep checks if the given absolute filename should be kept in the location history or not
func ShouldKeep(absFilename string) bool {
	if parentIsMan == nil {
		b := parentProcessIs("man")
		parentIsMan = &b
	}
	if *parentIsMan {
		return false
	}
	if strings.HasPrefix(absFilename, "/tmp/commit") || strings.HasPrefix(absFilename, "/tmp/man.") || strings.HasPrefix(absFilename, "/dev/") {
		// Not storing location info for files in /tmp or /dev
		return false
	}
	baseFilename := filepath.Base(absFilename)
	if strings.HasPrefix(baseFilename, "tmp.") || baseFilename == "-" {
		// Not storing location info for /tmp/tmp.* files or "-" files
		return false
	}
	if strings.HasPrefix(absFilename, "/tmp/") && !strings.Contains(baseFilename, ".") {
		// Not storing location info for files like /tmp/mutt-asdf-asdf-asdf-asdf
		return false
	}
	return true
}

// SaveLocation saves the current file position to the location history file
func (e *Editor) SaveLocation() error {
	// Save the current location in the location history and write it to file
	absFilename, err := e.AbsFilename()
	if err != nil {
		return err
	}
	return e.SaveLocationCustom(absFilename, locationHistory)
}

// SaveLocationCustom takes a filename (which includes the absolute path) and a map which contains
// an overview of which files were at which line location.
func (e *Editor) SaveLocationCustom(absFilename string, locationHistory LocationHistory) error {
	if locationHistory.Len() > maxLocationHistoryEntries {
		// Cull the history
		locationHistory = locationHistory.KeepNewest(maxLocationHistoryEntries)
	}

	if ShouldKeep(absFilename) {
		// Save the current line location
		locationHistory.Set(absFilename, e.LineNumber())
	}

	// Save the location history and return the error, if any
	return locationHistory.Save(locationHistoryFilename)
}

// KeepNewest removes all entries from the locationHistory except the N entries with the highest UNIX timestamp
func (locationHistory LocationHistory) KeepNewest(n int) LocationHistory {
	lenLocationHistory := len(locationHistory)
	if lenLocationHistory <= n {
		return locationHistory
	}

	keys := make([]int64, 0, lenLocationHistory)
	time2filename := make(map[int64]string)

	// Note that if there are timestamp collisions, the loss of rembembering a location in a file is acceptable.
	// Collisions are unlikely, though.

	for absFilename, lineNumberAndTimestamp := range locationHistory {
		timestamp := lineNumberAndTimestamp.Timestamp.Unix()
		keys = append(keys, timestamp)
		time2filename[timestamp] = absFilename
	}

	// Reverse sort
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	keys = keys[:n] // Keep only 'n' newest timestamps

	newLocationHistory := make(LocationHistory, n)
	for _, timestamp := range keys {
		absFilename := time2filename[timestamp]
		newLocationHistory[absFilename] = locationHistory[absFilename]
	}

	return newLocationHistory
}

// CloseLocksAndLocationHistory tries to close any active file locks and save the location history
func (e *Editor) CloseLocksAndLocationHistory(absFilename string, lockTimestamp time.Time, forceFlag bool, wg *sync.WaitGroup) {
	if canUseLocks.Load() {
		wg.Add(1)
		go func() {
			// Start by loading the lock overview, just in case something has happened in the mean time
			fileLock.Load()
			// Check if the lock is unchanged
			fileLockTimestamp := fileLock.GetTimestamp(absFilename)
			lockUnchanged := lockTimestamp == fileLockTimestamp
			// TODO: If the stored timestamp is older than uptime, unlock and save the lock overview
			if !forceFlag || lockUnchanged {
				// If the file has not been locked externally since this instance of the editor was loaded, don't
				// Unlock the current file and save the lock overview. Ignore errors because they are not critical.
				fileLock.Unlock(absFilename)
				fileLock.Save()
			}
			wg.Done()
		}()
	}
	// Save the current location in the location history and write it to file
	wg.Add(1)
	go func() {
		e.SaveLocationCustom(absFilename, locationHistory)
		wg.Done()
	}()
}
