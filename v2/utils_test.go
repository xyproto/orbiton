package main

import "testing"

func TestCapitalizeWords(t *testing.T) {
	if capitalizeWords("bob john") != "Bob John" {
		t.Fail()
	}
}
