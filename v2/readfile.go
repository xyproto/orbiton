package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/xyproto/binary"
)

// ReadFileAndProcessLines reads the named file concurrently, processes its lines, and updates the Editor.
func (e *Editor) ReadFileAndProcessLines(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if strings.HasSuffix(filename, ".gz") {
		data, err = gUnzipData(data)
		if err != nil {
			return err
		}
	}

	e.binaryFile = binary.Data(data)
	scanner := bufio.NewScanner(bytes.NewReader(data))

	const bufferSize = 512 * 1024

	// Set the scanner buffer size (max length per line)
	buf := make([]byte, bufferSize)
	scanner.Buffer(buf, bufferSize)

	lines := make(map[int][]rune)
	var index int
	var tabIndentCounter int64

	for scanner.Scan() {
		line := scanner.Text()
		if e.binaryFile {
			lines[index] = []rune(line)
		} else {
			line = opinionatedStringReplacer.Replace(line)
			if len(line) > 2 {
				var first byte = line[0]
				if first == '\t' {
					tabIndentCounter++
				} else if first == ' ' && line[1] == ' ' {
					tabIndentCounter--
				}
			}
			lines[index] = []rune(line)
		}
		index++
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		// most likely, this is a binary file and the lines are too long
		// TODO: Just read the file in another way and/or don't use a scanner
		return err
	}

	e.Clear()
	e.lines = lines

	if detectedTabs := tabIndentCounter > 0; !e.binaryFile {
		e.detectedTabs = &detectedTabs
		e.indentation.Spaces = !detectedTabs
	}

	e.changed = true

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
	// Prepare an empty map to load the lines into
	e.Clear()
	e.lines = make(map[int][]rune, 0)

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
	for index, byteLine := range byteLines {
		wg.Add(1)
		go e.LoadByteLine(IndexByteLine{index, byteLine}, &eMut, &tcMut, &tabIndentCounter, &numLines, &wg)
	}
	wg.Wait()

	// If the last line is empty, delete it
	if numLines > 0 && len(e.lines[numLines-1]) == 0 {
		delete(e.lines, numLines-1)
	}

	if detectedTabs := tabIndentCounter > 0; detectedTabs {
		// Check if there were more tab indentations than space indentations
		e.detectedTabs = &detectedTabs
		e.indentation.Spaces = !detectedTabs
	}

	// Mark the editor contents as "changed"
	e.changed = true
}
