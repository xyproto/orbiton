package main

import (
	"reflect"
	"testing"
)

func TestCapitalizeWords(t *testing.T) {
	if capitalizeWords("bob john") != "Bob John" {
		t.Fail()
	}
}

func TestWordWrap(t *testing.T) {
	text := `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.`
	maxWidth := 40
	expected := []string{
		"Lorem ipsum dolor sit amet, consectetur",
		"adipiscing elit. Sed do eiusmod tempor",
		"incididunt ut labore et dolore magna",
		"aliqua.",
	}

	result, err := wordWrap(text, maxWidth)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
