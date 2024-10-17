package main

import (
	"testing"
)

func TestParentIsMan(t *testing.T) {
	// This test will only pass if NOT run via the 'man' command (which is not typical.)
	if parentProcessIs("man") {
		t.Error("Parent process is 'man'.")
	} else {
		t.Logf("Parent process is not 'man'.")
	}
}
