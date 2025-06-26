package main

import (
	_ "embed"
	"errors"
	"regexp"
	"strings"

	"github.com/sajari/fuzzy"
	"github.com/xyproto/vt"
)

// how far away a word can be from the corrected word that the spellchecker suggests
const fuzzyDepth = 2

var (
	//go:embed words_en.txt.gz
	gzwords []byte

	spellChecker *SpellChecker

	errFoundNoTypos = errors.New("found no typos")
	wordRegexp      = regexp.MustCompile(`(?:%2[A-Z])?([a-zA-Z0-9]+)`) // avoid capturing "%2F" and "%2B", other than that, capture English words

	dontSuggest = []string{"urine"}
)

// SpellChecker is a slice of correct, custom and ignored words together with a *fuzzy.Model
type SpellChecker struct {
	fuzzyModel   *fuzzy.Model
	markedWord   string
	correctWords []string
	customWords  []string
	ignoredWords []string
}

// NewSpellChecker creates and initializes a new *SpellChecker.
// The embedded English word list is used to train the *fuzzy.Model.
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

// Train will train or re-train the current spellChecker.fuzzyModel, by using the current SpellChecker word slices
func (sc *SpellChecker) Train(reTrain bool) {
	if reTrain || sc.fuzzyModel == nil {

		// Initialize the spellchecker
		sc.fuzzyModel = fuzzy.NewModel()

		// This expands the distance searched, but costs more resources (memory and time).
		// For spell checking, "2" is typically enough, for query suggestions this can be higher
		sc.fuzzyModel.SetDepth(fuzzyDepth)

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
}

// CurrentSpellCheckWord returns the currently marked spell check word
func (e *Editor) CurrentSpellCheckWord() string {
	if spellChecker == nil {
		return ""
	}
	return spellChecker.markedWord
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

	var word string
	matches := wordRegexp.FindStringSubmatch(e.CurrentSpellCheckWord())
	if len(matches) > 1 { // Ensure that there's a captured group
		word = matches[1] // The captured word is in the second item of the slice
	}

	if hasS(spellChecker.customWords, word) || hasS(spellChecker.correctWords, word) { // already has this word
		return word
	}

	spellChecker.customWords = append(spellChecker.customWords, word)

	// Add the word
	spellChecker.fuzzyModel.TrainWord(word)

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

	var word string
	matches := wordRegexp.FindStringSubmatch(e.CurrentSpellCheckWord())
	if len(matches) > 1 { // Ensure that there's a captured group
		word = matches[1] // The captured word is in the second item of the slice
	}

	if hasS(spellChecker.ignoredWords, word) { // already has this word
		return word
	}
	spellChecker.ignoredWords = append(spellChecker.ignoredWords, word)

	spellChecker.Train(true) // re-train

	return word
}

// SearchForTypo returns the first misspelled word in the document (as defined by the dictionary),
// or an empty string. The second returned string is what the word could be if it was corrected.
func (e *Editor) SearchForTypo() (string, string, error) {
	if spellChecker == nil {
		newSpellChecker, err := NewSpellChecker()
		if err != nil {
			return "", "", err
		}
		spellChecker = newSpellChecker
	}
	e.spellCheckMode = true
	spellChecker.markedWord = ""

	// Use the regular expression to find all the words
	words := wordRegexp.FindAllString(e.String(), -1)

	// Now spellcheck all the words
	for _, word := range words {
		justTheWord := strings.TrimSpace(word)
		if justTheWord == "" {
			continue
		}
		if hasS(spellChecker.ignoredWords, justTheWord) || hasS(spellChecker.customWords, justTheWord) { // || hasS(spellChecker.correctWords, justTheWord) {
			continue
		}

		lower := strings.ToLower(justTheWord)

		if hasS(spellChecker.ignoredWords, lower) || hasS(spellChecker.customWords, lower) { // || hasS(spellChecker.correctWords, lower) {
			continue
		}

		corrected := spellChecker.fuzzyModel.SpellCheck(justTheWord)
		if !strings.EqualFold(justTheWord, corrected) && corrected != "" && !hasS(dontSuggest, corrected) { // case insensitive comparison of the original and spell-check-suggested word
			spellChecker.markedWord = justTheWord
			return justTheWord, corrected, nil
		}
	}
	return "", "", errFoundNoTypos
}

// NanoNextTypo tries to jump to the next typo
func (e *Editor) NanoNextTypo(c *vt.Canvas, status *StatusBar) (string, string) {
	if typo, corrected, err := e.SearchForTypo(); err == nil || err == errFoundNoTypos {
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
		if err == errFoundNoTypos || typo == "" {
			status.ClearAll(c, false)
			status.SetMessage("No typos found")
			status.Show(c, e)
			e.spellCheckMode = false
			e.ClearSearch()
			return "", ""
		}
		e.SetSearchTerm(c, status, typo, true) // true for spellCheckMode
		if err := e.GoToNextMatch(c, status, true, true); err == errNoSearchMatch {
			status.ClearAll(c, false)
			status.SetMessage("No more typos found")
			status.Show(c, e)
			e.spellCheckMode = false
			e.ClearSearch()
			return "", ""
		}
		if typo != "" && corrected != "" {
			status.ClearAll(c, false)
			status.SetMessage(typo + " could be " + corrected)
			status.Show(c, e)
		}
		return typo, corrected
	}
	return "", ""
}
