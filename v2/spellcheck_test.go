package main

import (
	"testing"
)

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

func TestNoTypo(t *testing.T) {
	e := NewSimpleEditor(80)
	e.InsertStringAndMove(nil, "This is a correct sentence.")

	_, _, err := e.SearchForTypo()
	if err != errFoundNoTypos {
		t.Fatal("Expected 'errFoundNoTypos' when searching for typo in a correct sentence.")
	}
}
