package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"runtime"
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
	e.binaryFile = binary.DataAccurate(data)
	if !e.binaryFile {
		data = []byte(opinionatedStringReplacer.Replace(string(data)))
	}

	var (
		reader           = bufio.NewReader(bytes.NewReader(data))
		lines            = make(map[int][]rune)
		index            int
		tabIndentCounter int64
		minSpaces        int
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
			if len(line) > 2 {
				first = line[0]
				if first == '\t' {
					tabIndentCounter++
				} else if first == ' ' && line[1] == ' ' {
					tabIndentCounter--
					// Count leading spaces to detect indentation width
					spaces := 0
					for i := 0; i < len(line) && line[i] == ' '; i++ {
						spaces++
					}
					if spaces >= 2 && (minSpaces == 0 || spaces < minSpaces) {
						minSpaces = spaces
					}
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
	if !e.binaryFile {
		if detectedTabs := tabIndentCounter > 0; detectedTabs {
			e.detectedTabs = &detectedTabs
			e.indentation.Spaces = false
		} else if tabIndentCounter < 0 {
			detectedTabs := false
			e.detectedTabs = &detectedTabs
			e.indentation.Spaces = true
			if minSpaces >= 2 {
				e.indentation.PerTab = minSpaces
			}
		}
	}
	e.MarkChanged()
	return nil
}

// LoadByteLine loads a single byte line
func (e *Editor) LoadByteLine(ib IndexByteLine, eMut, tcMut *sync.RWMutex, tabIndentCounter, minSpaces, numLines *int) {
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
			// Count leading spaces to detect indentation width
			spaces := 0
			for i := 0; i < len(ib.byteLine) && ib.byteLine[i] == ' '; i++ {
				spaces++
			}
			if spaces >= 2 && (*minSpaces == 0 || spaces < *minSpaces) {
				*minSpaces = spaces
			}
			tcMut.Unlock()
		}
	}
	eMut.Lock()
	e.lines[ib.index] = bytes.Runes(ib.byteLine)
	*numLines++
	eMut.Unlock()
}

// LoadBytes replaces the current editor contents with the given bytes
func (e *Editor) LoadBytes(data []byte) {
	e.binaryFile = binary.DataAccurate(data)
	if !e.binaryFile {
		data = []byte(opinionatedStringReplacer.Replace(string(data)))
	}

	lineCount := bytes.Count(data, []byte{'\n'}) + 1

	var (
		// Split the bytes into lines
		byteLines = bytes.Split(data, []byte{'\n'})

		// Place the lines into the editor, while counting tab indentations vs space indentations
		tabIndentCounter int

		// Minimum leading space count for detecting spaces-per-tab
		minSpaces int

		// Count the number of lines as the lines are being processed
		numLines int

		// Mutex for the editor lines and the numLines counter
		eMut sync.RWMutex

		// Mutex for the tabIndentCounter and minSpaces
		tcMut sync.RWMutex
	)

	workerCount := max(runtime.GOMAXPROCS(0), 1)
	jobs := make(chan IndexByteLine, workerCount*2)
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for ib := range jobs {
				e.LoadByteLine(ib, &eMut, &tcMut, &tabIndentCounter, &minSpaces, &numLines)
			}
		}()
	}
	for i := range lineCount {
		jobs <- IndexByteLine{byteLines[i], i}
	}
	close(jobs)
	wg.Wait()

	// If the last line is empty, delete it
	if numLines > 0 && len(e.lines[numLines-1]) == 0 {
		delete(e.lines, numLines-1)
	}

	if detectedTabs := tabIndentCounter > 0; detectedTabs {
		// More tab indentations than space indentations
		e.detectedTabs = &detectedTabs
		e.indentation.Spaces = false
	} else if tabIndentCounter < 0 {
		// More space indentations than tab indentations
		detectedTabs := false
		e.detectedTabs = &detectedTabs
		e.indentation.Spaces = true
		if minSpaces >= 2 {
			e.indentation.PerTab = minSpaces
		}
	}

	// Mark the editor contents as "changed"
	e.MarkChanged()
}
