package main

import (
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
	typos, corrected, err := e.SearchForTypo(nil, nil)
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
