package main

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"unicode"

	"github.com/xyproto/binary"
)

func (e *Editor) ReadAllLinesConcurrently(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if strings.HasSuffix(filename, ".gz") {
		data, err = gUnzipData(data)
		if err != nil {
			return err
		}
	}

	isBinary := binary.Data(data)
	scanner := bufio.NewScanner(bytes.NewReader(data))

	var (
		lines            sync.Map
		wg               sync.WaitGroup
		tabIndentCounter int
		tcMut            = &sync.Mutex{}
	)

	processLine := func(index int, line string) {
		lines.Store(index, []rune(line))
		wg.Done()
	}

	processLineWithOpinion := func(index int, line string) {
		line = opinionatedStringReplacer.Replace(line)
		line = strings.TrimRightFunc(line, unicode.IsSpace)

		if len(line) > 2 {
			var first byte = line[0]
			if first == '\t' {
				tcMut.Lock()
				tabIndentCounter++
				tcMut.Unlock()
			} else if first == ' ' && line[1] == ' ' {
				tcMut.Lock()
				tabIndentCounter--
				tcMut.Unlock()
			}
		}

		lines.Store(index, []rune(line))
		wg.Done()
	}

	var index int
	for scanner.Scan() {
		line := scanner.Text()
		wg.Add(1)
		if isBinary {
			go processLine(index, line)
		} else {
			go processLineWithOpinion(index, line)
		}
		index++
	}

	wg.Wait()

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}

	e.Clear()
	e.lines = make(map[int][]rune, index)

	lines.Range(func(key, value interface{}) bool {
		e.lines[key.(int)] = value.([]rune)
		return true
	})

	// Set e.detectedTabs and e.indentation.Spaces
	if detectedTabs := tabIndentCounter > 0; !isBinary {
		e.detectedTabs = &detectedTabs
		e.indentation.Spaces = !detectedTabs
	}

	e.changed = true

	return nil
}
