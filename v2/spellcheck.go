package main

import (
	_ "embed"
	"strings"

	"github.com/sajari/fuzzy"
)

//go:embed words.txt.gz
var gzwords []byte

func (e *Editor) SpellCheck() error {

	wordData, err := gUnzipData(gzwords)
	if err != nil {
		return err
	}

	spellcheckWords := strings.Fields(string(wordData))

	// Initialize the spellchecker
	model := fuzzy.NewModel()

	// For testing only, this is not advisable on production
	model.SetThreshold(1)

	// This expands the distance searched, but costs more resources (memory and time).
	// For spell checking, "2" is typically enough, for query suggestions this can be higher
	model.SetDepth(5)

	// Train multiple words simultaneously by passing an array of strings to the "Train" function
	model.Train(spellcheckWords)

	// Now spellcheck all the words, and log the results
	for _, word := range strings.Fields(e.String()) {
		corrected := model.SpellCheck(word)
		if word != corrected {
			logf("%s is wrong\n", word)
		}
	}

	return nil
}
