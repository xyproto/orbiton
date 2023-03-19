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

// bufferSize is the max length per line when reading files
const bufferSize = 64 * 1024

// ReadFile reads in a file, concurrently
func ReadFile(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data := make([]byte, 0)
	chunks := make(chan []byte)
	errors := make(chan error)
	done := make(chan struct{})
	var wg sync.WaitGroup
	go func() {
		defer close(done)
		for {
			wg.Add(1)
			go func() {
				defer wg.Done()
				buf := make([]byte, bufferSize)
				n, err := f.Read(buf)
				if n > 0 {
					chunks <- buf[:n]
				}
				if err != nil {
					errors <- err
					return
				}
			}()
			select {
			case chunk := <-chunks:
				data = append(data, chunk...)
			case err := <-errors:
				if err == io.EOF {
					err = nil
				}
				wg.Wait()
				done <- struct{}{}
				return
			}
		}
	}()
	<-done
	return data, nil
}

// ReadFileAndProcessLines reads the named file concurrently, processes its lines, and updates the Editor.
func (e *Editor) ReadFileAndProcessLines(filename string) error {
	data, err := ReadFile(filename)
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
