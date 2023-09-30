package main

import (
	"testing"
)

func TestParentIsMan(t *testing.T) {
	// This test will only pass if run from the 'man' command, which is not typical.
	isMan := parentIsMan()
	if isMan {
		t.Logf("Parent process is 'man'.")
	} else {
		t.Logf("Parent process is not 'man'.")
	}
}
