package main

import (
	"strings"
	"testing"
)

func TestAddAndRemoveWord(t *testing.T) {
	e := NewSimpleEditor(80)

	// Setup the Editor content
	e.InsertStringAndMove(nil, "testword")
	e.Home()
	word := e.CurrentWord()

	if word == "" {
		t.Fatal("Word is empty!")
	}

	// Test Add
	addedWord := e.AddCurrentWordToWordList()
	if addedWord != word {
		t.Fatalf("Expected word \"%s\" to be added, but got \"%s\".", word, addedWord)
	}
	if !hasS(spellChecker.customWords, word) {
		t.Fatalf("Word %s was not added to custom words.", word)
	}

	// Test Remove
	removedWord := e.RemoveCurrentWordFromWordList()
	if removedWord != word {
		t.Fatalf("Expected word \"%s\" to be removed, but got \"%s\".", word, removedWord)
	}
	if !hasS(spellChecker.ignoredWords, word) {
		t.Fatalf("Word %s was not added to ignored words.", word)
	}
}

func TestSearchForTypo(t *testing.T) {
	e := NewSimpleEditor(80)

	// Setup the Editor content with a typo
	e.InsertStringAndMove(nil, "helllo world")
	// Assuming "helllo" is a typo
	typos, corrected, err := e.SearchForTypo()
	if err != nil {
		t.Fatalf("Error encountered when searching for typo: %s", err)
	}
	if typos != "helllo" {
		t.Fatalf("Expected typo \"helllo\" but got \"%s\".", typos)
	}
	// Assuming the correction for "helllo" is "hello"
	if corrected != "hello" {
		t.Fatalf("Expected corrected word \"hello\" but got \"%s\".", corrected)
	}
}

func TestSpellCheckerTraining(t *testing.T) {
	sc, _ := NewSpellChecker()
	initialModel := sc.fuzzyModel
	sc.Train(true) // force re-train
	if sc.fuzzyModel == initialModel {
		t.Fatal("Fuzzy model did not retrain as expected.")
	}
}

func TestDefaultSpellCheckerInitialization(t *testing.T) {
	if spellChecker == nil {
		t.Fatal("Default spell checker is not initialized.")
	}
	if len(spellChecker.correctWords) == 0 {
		t.Fatal("Default spell checker has no correct words loaded.")
	}
}

func TestMultipleTypos(t *testing.T) {
	e := NewSimpleEditor(80)
	e.InsertStringAndMove(nil, "Thiss is a testt sentencee with multiplee typos.")

	expectedTypos := []string{"Thiss", "testt", "sentencee", "multiplee"}
	for _, expectedTypo := range expectedTypos {
		typo, _, err := e.SearchForTypo()
		if err != nil {
			t.Fatalf("Error encountered when searching for typo: %s", err)
		}
		if typo != expectedTypo {
			t.Fatalf("Expected typo \"%s\" but got \"%s\".", expectedTypo, typo)
		}
		// Simulate correcting the typo to continue the loop
		e.InsertStringAndMove(nil, strings.Repeat("\b", len(typo)))
		e.InsertStringAndMove(nil, " ")
	}
}

func TestNoTypo(t *testing.T) {
	e := NewSimpleEditor(80)
	e.InsertStringAndMove(nil, "This is a correct sentence.")

	_, _, err := e.SearchForTypo()
	if err != errFoundNoTypos {
		t.Fatal("Expected 'errFoundNoTypos' when searching for typo in a correct sentence.")
	}
}
