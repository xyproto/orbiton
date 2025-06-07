package main

import (
	"bytes"
	"os"
	"strings"
	"sync"

	"github.com/xyproto/binary"
)

// ReadFileAndProcessLines reads the named file concurrently, processes its lines, and updates the Editor.
func (e *Editor) ReadFileAndProcessLines(filename string) error {
	// Try to memory-map the file for faster loading; fall back to ReadFile on error.
	data, unmap, err := mmapFile(filename)
	if err != nil {
		data, err = os.ReadFile(filename)
		if err != nil {
			return err
		}
	} else {
		defer unmap()
	}
	// Decompress .gz files if needed
	if strings.HasSuffix(filename, ".gz") {
		data, err = gUnzipData(data)
		if err != nil {
			return err
		}
	}
	e.binaryFile = binary.Data(data)

	// For text files, apply the opinionated replacer once to the entire data.
	if !e.binaryFile {
		data = []byte(opinionatedStringReplacer.Replace(string(data)))
	}

	rawLines := bytes.Split(data, []byte{'\n'})
	lines := make(map[int][]rune, len(rawLines))
	var tabIndentCounter int64
	for i, raw := range rawLines {
		if len(raw) > 2 {
			if raw[0] == '\t' {
				tabIndentCounter++
			} else if raw[0] == ' ' && raw[1] == ' ' {
				tabIndentCounter--
			}
		}
		runes := make([]rune, len(raw))
		for j, b := range raw {
			runes[j] = rune(b)
		}
		lines[i] = runes
	}

	e.Clear()

	e.lines = lines
	if detectedTabs := tabIndentCounter > 0; !e.binaryFile && e.indentation.Spaces {
		e.detectedTabs = &detectedTabs
		e.indentation.Spaces = !detectedTabs
	}
	e.changed.Store(true)
	return nil
}

// LoadByteLine loads a single byte line
func (e *Editor) LoadByteLine(ib IndexByteLine, eMut, tcMut *sync.RWMutex, tabIndentCounter, numLines *int, wg *sync.WaitGroup) {
	// Require at least two bytes. Ignore lines with a single tab indentation or a single space
	if len(ib.byteLine) > 2 {
		first := ib.byteLine[0]
		if first == '\t' {
			tcMut.Lock()
			*tabIndentCounter++ // a tab indentation counts like a positive tab indentation
			tcMut.Unlock()
		} else if first == ' ' && ib.byteLine[1] == ' ' { // assume that two spaces is the smallest space indentation
			tcMut.Lock()
			*tabIndentCounter-- // a space indentation counts like a negative tab indentation
			tcMut.Unlock()
		}
	}
	eMut.Lock()
	e.lines[ib.index] = []rune(string(ib.byteLine))
	*numLines++
	eMut.Unlock()
	wg.Done()
}

// LoadBytes replaces the current editor contents with the given bytes
func (e *Editor) LoadBytes(data []byte) {
	lineCount := bytes.Count(data, []byte{'\n'}) + 1

	// Prepare an empty map to load the lines into
	e.Clear()
	e.lines = make(map[int][]rune, lineCount)

	e.binaryFile = binary.Data(data)

	var (
		// Split the bytes into lines
		byteLines = bytes.Split(data, []byte{'\n'})

		// Place the lines into the editor, while counting tab indentations vs space indentations
		tabIndentCounter int

		// Count the number of lines as the lines are being processed
		numLines int

		// Mutex for the editor lines and the numLines counter
		eMut sync.RWMutex

		// Mutex for the tabIndentCounter
		tcMut sync.RWMutex
	)

	var wg sync.WaitGroup
	var byteLine []byte
	for i := 0; i < lineCount; i++ {
		byteLine = byteLines[i]
		wg.Add(1)
		go e.LoadByteLine(IndexByteLine{byteLine, i}, &eMut, &tcMut, &tabIndentCounter, &numLines, &wg)
	}
	wg.Wait()

	// If the last line is empty, delete it
	if numLines > 0 && len(e.lines[numLines-1]) == 0 {
		delete(e.lines, numLines-1)
	}

	if detectedTabs := tabIndentCounter > 0; detectedTabs && e.indentation.Spaces {
		// Check if there were more tab indentations than space indentations
		e.detectedTabs = &detectedTabs
		e.indentation.Spaces = !detectedTabs
	}

	// Mark the editor contents as "changed"
	e.changed.Store(true)
}
