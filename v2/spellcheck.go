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

	spellChecker *SpellChecker

	errFoundNoTypos    = errors.New("found no typos")
	letterDigitsRegexp = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
)

type SpellChecker struct {
	correctWords []string
	customWords  []string
	ignoredWords []string
	fuzzyModel   *fuzzy.Model
}

func NewSpellChecker() (*SpellChecker, error) {
	var sc SpellChecker

	sc.customWords = make([]string, 0)
	sc.ignoredWords = make([]string, 0)

	wordData, err := gUnzipData(gzwords)
	if err != nil {
		return nil, err
	}
	sc.correctWords = strings.Fields(string(wordData))

	sc.Train(false) // training for the first time, not re-training

	return &sc, nil
}

func (sc *SpellChecker) Train(reTrain bool) {
	if reTrain || sc.fuzzyModel == nil {

		// Initialize the spellchecker
		sc.fuzzyModel = fuzzy.NewModel()

		// This expands the distance searched, but costs more resources (memory and time).
		// For spell checking, "2" is typically enough, for query suggestions this can be higher
		sc.fuzzyModel.SetDepth(2)

		lenCorrect := len(sc.correctWords)
		lenCustom := len(sc.customWords)

		trainWords := make([]string, lenCorrect+lenCustom) // initialize with enough capacity

		var word string

		for i := 0; i < lenCorrect; i++ {
			word := sc.correctWords[i]
			if !hasS(sc.ignoredWords, word) {
				trainWords = append(trainWords, word)
			}
		}

		for i := 0; i < lenCustom; i++ {
			word = sc.customWords[i]
			if !hasS(sc.ignoredWords, word) {
				trainWords = append(trainWords, word)
			}
		}

		// Train multiple words simultaneously by passing an array of strings to the "Train" function
		sc.fuzzyModel.Train(trainWords)
	}

	return
}

// AddCurrentWordToWordList will attempt to add the word at the cursor to the spellcheck word list
func (e *Editor) AddCurrentWordToWordList() string {
	if spellChecker == nil {
		newSpellChecker, err := NewSpellChecker()
		if err != nil {
			return ""
		}
		spellChecker = newSpellChecker
	}

	word := strings.TrimSpace(letterDigitsRegexp.ReplaceAllString(e.CurrentWord(), ""))

	if hasS(spellChecker.customWords, word) || hasS(spellChecker.correctWords, word) { // already has this word
		return ""
	}

	spellChecker.customWords = append(spellChecker.customWords, word)

	spellChecker.Train(true) // re-train

	return word
}

// RemoveCurrentWordFromWordList will attempt to add the word at the cursor to the spellcheck word list
func (e *Editor) RemoveCurrentWordFromWordList() string {
	if spellChecker == nil {
		newSpellChecker, err := NewSpellChecker()
		if err != nil {
			return ""
		}
		spellChecker = newSpellChecker
	}

	word := strings.TrimSpace(letterDigitsRegexp.ReplaceAllString(e.CurrentWord(), ""))

	if hasS(spellChecker.ignoredWords, word) { // already has this word
		return ""
	}
	spellChecker.ignoredWords = append(spellChecker.ignoredWords, word)

	spellChecker.Train(true) // re-train

	return word
}

// SearchForTypo returns the first misspelled word in the document (as defined by the dictionary),
// or an empty string. The second returned string is what the word could be if it was corrected.
func (e *Editor) SearchForTypo(c *vt100.Canvas, status *StatusBar) (string, string, error) {
	if spellChecker == nil {
		newSpellChecker, err := NewSpellChecker()
		if err != nil {
			return "", "", err
		}
		spellChecker = newSpellChecker
	}

	e.spellCheckMode = true

	// Now spellcheck all the words, and log the results
	for _, word := range strings.Fields(e.String()) {
		// Remove special characters
		justTheWord := strings.TrimSpace(letterDigitsRegexp.ReplaceAllString(word, ""))
		if justTheWord == "" {
			continue
		}
		if hasS(spellChecker.ignoredWords, justTheWord) || hasS(spellChecker.correctWords, justTheWord) {
			continue
		}

		if corrected := spellChecker.fuzzyModel.SpellCheck(justTheWord); word != corrected {
			return justTheWord, corrected, nil
		}
	}

	return "", "", errFoundNoTypos
}

// NanoNextTypo tries to jump to the next typo
func (e *Editor) NanoNextTypo(c *vt100.Canvas, status *StatusBar) {
	if typo, corrected, err := e.SearchForTypo(c, status); err == nil || err == errFoundNoTypos {
		e.redraw = true
		e.redrawCursor = true
		if err == errFoundNoTypos || typo == "" {
			status.ClearAll(c)
			status.SetMessage("No typos found")
			status.Show(c, e)
			return
		}
		if typo != "" && corrected != "" {
			status.ClearAll(c)
			status.SetMessage(typo + " could be " + corrected)
			status.Show(c, e)
			return
		}
		e.SetSearchTerm(c, status, typo, true) // true for spellCheckMode
		if err := e.GoToNextMatch(c, status, true, true); err == errNoSearchMatch {
			e.ClearSearch()
			status.ClearAll(c)
			status.SetMessage("No typos found")
			status.Show(c, e)
			return
		}
		if typo != "" && corrected != "" {
			status.ClearAll(c)
			status.SetMessage(typo + " could be " + corrected)
			status.Show(c, e)
		}
	}
}
