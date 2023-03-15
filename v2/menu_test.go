package main

import (
	"testing"
)

func TestLetterInSmallWord(t *testing.T) {
	s := "hello in there"
	if inSmallWordOrSpace(s, 0) { // "h"
		// fmt.Println(0)
		t.Fail()
	}
	if !inSmallWordOrSpace(s, 5) { // " "
		// fmt.Println(5)
		t.Fail()
	}
	if !inSmallWordOrSpace(s, 6) { // "i"
		// fmt.Println(6)
		t.Fail()
	}
	if !inSmallWordOrSpace(s, 7) { // "n"
		// fmt.Println(7)
		t.Fail()
	}
	if !inSmallWordOrSpace(s, 8) { // " "
		// fmt.Println(8)
		t.Fail()
	}
	if inSmallWordOrSpace(s, 9) { // "t"
		// fmt.Println(9)
		t.Fail()
	}
}

func TestSelectionLettersForChoices(_ *testing.T) {
	choices := []string{
		"This is a string",
		"This is another",
		"This is yet a string",
		"And here is another",
	}
	counter := 0
	for _, stringAndPosition := range selectionLettersForChoices(choices) {
		s := choices[counter]
		counter++
		runes := []rune(s)
		runes[stringAndPosition.pos] = '!'
		s = string(runes)
		// fmt.Println(s)
		_ = s
	}
}
