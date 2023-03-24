package main

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/xyproto/binary"
)

const chunkSize = 64 * 1024 // size of chunks to read

func readChunks(reader io.Reader, chunkChan chan<- []byte, errorChan chan<- error) {
	defer close(chunkChan)
	defer close(errorChan)

	buf := make([]byte, chunkSize)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			chunkChan <- chunk
		}
		if err != nil {
			if err != io.EOF {
				errorChan <- err
			}
			break
		}
	}
}

// ReadFile reads in a file concurrently
func ReadFile(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	chunkChan := make(chan []byte)
	errorChan := make(chan error)

	go readChunks(f, chunkChan, errorChan)

	var data bytes.Buffer
	var leftover []byte

	for chunk := range chunkChan {
		if len(leftover) > 0 {
			chunk = append(leftover, chunk...)
			leftover = nil
		}
		lastNewLine := bytes.LastIndex(chunk, []byte("\n"))
		if lastNewLine != -1 {
			leftover = chunk[lastNewLine+1:]
			chunk = chunk[:lastNewLine+1]
		}
		data.Write(chunk)
	}

	if len(leftover) > 0 {
		data.Write(leftover)
	}

	if err := <-errorChan; err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

// splitAtNewline splits the given byte slice at the first newline character
// and returns the parts before and after the newline (excluding the newline character itself)
func splitAtNewline(data []byte) (before, after []byte) {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return data[:i], data[i+1:]
	}
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

	lines := make(map[int][]rune)
	var index int
	var tabIndentCounter int64

	for {
		line, rest := splitAtNewline(data)
		if len(line) > 0 {
			if e.binaryFile {
				lines[index] = []rune(string(line))
			} else {
				lineStr := opinionatedStringReplacer.Replace(string(line))
				if len(lineStr) > 2 {
					var first byte = lineStr[0]
					if first == '\t' {
						tabIndentCounter++
					} else if first == ' ' && lineStr[1] == ' ' {
						tabIndentCounter--
					}
				}
				lines[index] = []rune(lineStr)
			}
			index++
		}
		if len(rest) == 0 {
			break
		}
		data = rest
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
