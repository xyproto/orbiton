package main

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/xyproto/mode"
)

// FindFunctionSignatures tries to find all function signatures relevant to this project.
// Currently, only Go is supported, and it only searches the current directory.
func (e *Editor) FindFunctionSignatures() []string {
	ext := ".go"

	wordCount := make(map[string]int)

	filenames, err := filepath.Glob("*" + ext)
	if err != nil {
		return []string{}
	}

	var re *regexp.Regexp
	switch e.mode {
	case mode.Go:
		re = regexp.MustCompile(`(func .*) +{+`)
	default:
		return []string{}
	}

	var data []byte
	var highestCount int
	for _, filename := range filenames {
		data, err = os.ReadFile(filename)
		if err != nil {
			continue
		}
		submatches := re.FindAllStringSubmatch(string(data), -1)
		for _, submatch := range submatches {
			word := submatch[1]
			if _, ok := wordCount[word]; ok {
				wordCount[word]++
				if wordCount[word] > highestCount {
					highestCount = wordCount[word]
				}
			} else {
				wordCount[word] = 1
				if wordCount[word] > highestCount {
					highestCount = wordCount[word]
				}
			}
		}
	}

	// Copy the words from the map to a string slice, such
	// that the most frequent words appear first.
	sl := make([]string, len(wordCount))
	slIndex := 0
	for i := highestCount; i >= 0; i-- {
		for word, count := range wordCount {
			if count == i && len(word) > 0 {
				sl[slIndex] = word
				slIndex++
			}
		}
	}

	return sl
}
