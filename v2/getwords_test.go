package main

import (
	"testing"
)

func TestGetWords(_ *testing.T) {
	s := "words can be colored, this is all gray"
	GetWords(s, "lightblue", "red", "lightgreen", "yellow", "darkgray")
}
