package main

import (
	_ "embed"
	"errors"
	"regexp"
	"strings"

	"github.com/sajari/fuzzy"
	"github.com/xyproto/vt100"
)

var (
	//go:embed english_word_list.txt.gz
	gzwords []byte

	fuzzyModel         *fuzzy.Model
	correctWords       []string
	errFoundNoTypos    = errors.New("found no typos")
	letterDigitsRegexp = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
)

func initSpellcheck() {
	if len(correctWords) == 0 {
		wordData, err := gUnzipData(gzwords)
		if err != nil {
			return
		}
		correctWords = strings.Fields(string(wordData))
	}
	if fuzzyModel == nil {

		// Initialize the spellchecker
		fuzzyModel = fuzzy.NewModel()

		// For testing only, this is not advisable on production
		//model.SetThreshold(1)

		// This expands the distance searched, but costs more resources (memory and time).
		// For spell checking, "2" is typically enough, for query suggestions this can be higher
		fuzzyModel.SetDepth(2)

		// Train multiple words simultaneously by passing an array of strings to the "Train" function
		fuzzyModel.Train(correctWords)
	}
}

// AddCurrentWordToWordList will attempt to add the word at the cursor to the spellcheck word list
func (e *Editor) AddCurrentWordToWordList() string {
	initSpellcheck()

	word := strings.TrimSpace(letterDigitsRegexp.ReplaceAllString(e.CurrentWord(), ""))
	if hasS(correctWords, word) { // already has this word
		return ""
	}
	correctWords = append(correctWords, word)

	fuzzyModel = fuzzy.NewModel()
	fuzzyModel.SetDepth(2)
	fuzzyModel.Train(correctWords)
	return word
}

// RemoveCurrentWordFromWordList will attempt to add the word at the cursor to the spellcheck word list
func (e *Editor) RemoveCurrentWordFromWordList() string {
	initSpellcheck()

	word := strings.TrimSpace(letterDigitsRegexp.ReplaceAllString(e.CurrentWord(), ""))

	l := len(correctWords)

	if l == 0 { // can not remove from an empty list
		return ""
	}

	wordIndex := -1
	for i := 0; i < l; i++ {
		if correctWords[i] == word {
			wordIndex = i
			break
		}
	}
	if wordIndex == -1 { // not found
		return ""
	}

	lastIndex := l - 1
	correctWords[wordIndex] = correctWords[lastIndex]
	correctWords = correctWords[:lastIndex]

	fuzzyModel = fuzzy.NewModel()
	fuzzyModel.SetDepth(2)
	fuzzyModel.Train(correctWords)

	return word
}

// SearchForTypo returns the first misspelled word in the document (as defined by the dictionary),
// or an empty string.
func (e *Editor) SearchForTypo(c *vt100.Canvas, status *StatusBar) (string, error) {
	initSpellcheck()

	e.spellCheckMode = true

	// Now spellcheck all the words, and log the results
	for _, word := range strings.Fields(e.String()) {
		// Remove special characters
		justTheWord := strings.TrimSpace(letterDigitsRegexp.ReplaceAllString(word, ""))
		if justTheWord == "" {
			continue
		}
		if hasS(correctWords, justTheWord) {
			continue
		}

		if corrected := fuzzyModel.SpellCheck(justTheWord); word != corrected {
			status.Clear(c)
			status.SetMessage(justTheWord + " could be " + corrected)
			status.Show(c, e)
			return justTheWord, nil
		}
	}

	return "", errFoundNoTypos
}
