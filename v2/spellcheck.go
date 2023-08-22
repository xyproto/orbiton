package main

import (
	_ "embed"
	"regexp"
	"strings"

	"github.com/sajari/fuzzy"
	"github.com/xyproto/vt100"
)

//go:embed words.txt.gz
var gzwords []byte

// SpellCheck checks every word compared to the embedded word list
func (e *Editor) SpellCheck(c *vt100.Canvas, status *StatusBar) error {

	wordData, err := gUnzipData(gzwords)
	if err != nil {
		return err
	}

	spellcheckWords := strings.Fields(string(wordData))

	// TODO: Figure out what hangs

	// Initialize the spellchecker
	model := fuzzy.NewModel()

	// For testing only, this is not advisable on production
	//model.SetThreshold(1)

	// This expands the distance searched, but costs more resources (memory and time).
	// For spell checking, "2" is typically enough, for query suggestions this can be higher
	model.SetDepth(2)

	// Train multiple words simultaneously by passing an array of strings to the "Train" function
	model.Train(spellcheckWords)

	//status.Clear(c)
	//status.SetMessage(fmt.Sprintf("%d words", len(spellcheckWords)))
	//status.Show(c, e)
	//return nil

	re := regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

	// Now spellcheck all the words, and log the results
	foundTypo := false
	for _, word := range strings.Fields(e.String()) {
		// Remove special characters
		justTheWord := re.ReplaceAllString(word, "")
		if justTheWord == "" {
			continue
		}
		corrected := model.SpellCheck(justTheWord)
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
