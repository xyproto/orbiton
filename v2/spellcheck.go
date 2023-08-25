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

	fuzzyModel      *fuzzy.Model
	correctWords    []string
	errFoundNoTypos = errors.New("found no typos")
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
func (e *Editor) AddCurrentWordToWordList(c *vt100.Canvas, status *StatusBar) {
	initSpellcheck()

	re := regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
	word := re.ReplaceAllString(e.CurrentWord(), "")

	if !hasS(correctWords, word) {
		correctWords = append(correctWords, word)
	}

	fuzzyModel = fuzzy.NewModel()
	fuzzyModel.SetDepth(2)
	fuzzyModel.Train(correctWords)

	status.Clear(c)
	status.SetMessage("Added " + word)
	status.Show(c, e)
}

// SearchForTypo returns the first misspelled word in the document (as defined by the dictionary),
// or an empty string.
func (e *Editor) SearchForTypo(c *vt100.Canvas, status *StatusBar) (string, error) {
	initSpellcheck()

	re := regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

	// Now spellcheck all the words, and log the results
	for _, word := range strings.Fields(e.String()) {
		// Remove special characters
		justTheWord := re.ReplaceAllString(word, "")
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

// SpellCheck checks every word compared to the embedded word list
// TODO: Introduce a callback function
func (e *Editor) SpellCheck(c *vt100.Canvas, status *StatusBar) error {

	initSpellcheck()

	re := regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

	// Now spellcheck all the words, and log the results
	foundTypo := false
	for _, word := range strings.Fields(e.String()) {
		// Remove special characters
		justTheWord := re.ReplaceAllString(word, "")
		if justTheWord == "" {
			continue
		}
		corrected := fuzzyModel.SpellCheck(justTheWord)
		if word != corrected {

			// TODO: Ask the user if the word should be learned or ignored,
			//       then keep this state in the cache.

			status.Clear(c)
			status.SetMessage(justTheWord + " should be " + corrected + "?")
			status.Show(c, e)
			foundTypo = true
			break
		}
	}

	if !foundTypo {
		status.Clear(c)
		status.SetMessage("found no typos")
		status.Show(c, e)
	}

	return nil
}
