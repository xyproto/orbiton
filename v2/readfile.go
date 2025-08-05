package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"

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

	var (
		reader           = bufio.NewReader(bytes.NewReader(data))
		lines            = make(map[int][]rune)
		index            int
		tabIndentCounter int64
		first            byte
	)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		// Remove the newline character at the end of the line
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		if e.binaryFile {
			lines[index] = []rune(line)
		} else {
			line = opinionatedStringReplacer.Replace(line)
			if len(line) > 2 {
				first = line[0]
				if first == '\t' {
					tabIndentCounter++
				} else if first == ' ' && line[1] == ' ' {
					tabIndentCounter--
				}
			}
			lines[index] = []rune(line)
		}
		index++

		if err == io.EOF {
			break
		}
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
func (e *Editor) LoadByteLine(ib IndexByteLine, eMut *sync.RWMutex, tabIndentCounter, numLines *int64, wg *sync.WaitGroup) {
	// Require at least two bytes. Ignore lines with a single tab indentation or a single space
	if len(ib.byteLine) > 2 {
		first := ib.byteLine[0]
		if first == '\t' {
			atomic.AddInt64(tabIndentCounter, 1) // a tab indentation counts like a positive tab indentation
		} else if first == ' ' && ib.byteLine[1] == ' ' { // assume that two spaces is the smallest space indentation
			atomic.AddInt64(tabIndentCounter, -1) // a space indentation counts like a negative tab indentation
		}
	}
	eMut.Lock()
	e.lines[ib.index] = []rune(string(ib.byteLine))
	eMut.Unlock()
	atomic.AddInt64(numLines, 1)
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
		tabIndentCounter int64

		// Count the number of lines as the lines are being processed
		numLines int64

		// Mutex for the editor lines
		eMut sync.RWMutex
	)

	var wg sync.WaitGroup
	var byteLine []byte
	for i := 0; i < lineCount; i++ {
		byteLine = byteLines[i]
		wg.Add(1)
		go e.LoadByteLine(IndexByteLine{byteLine, i}, &eMut, &tabIndentCounter, &numLines, &wg)
	}
	wg.Wait()

	// If the last line is empty, delete it
	numLinesVal := atomic.LoadInt64(&numLines)
	if numLinesVal > 0 && len(e.lines[int(numLinesVal-1)]) == 0 {
		delete(e.lines, int(numLinesVal-1))
	}

	tabIndentCounterVal := atomic.LoadInt64(&tabIndentCounter)
	if detectedTabs := tabIndentCounterVal > 0; detectedTabs && e.indentation.Spaces {
		// Check if there were more tab indentations than space indentations
		e.detectedTabs = &detectedTabs
		e.indentation.Spaces = !detectedTabs
	}

	// Mark the editor contents as "changed"
	e.changed.Store(true)
}
