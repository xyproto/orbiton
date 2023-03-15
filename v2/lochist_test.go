package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestVimInfo(_ *testing.T) {
	LoadVimLocationHistory(vimLocationHistoryFilename)
}

func TestEmacsPlaces(_ *testing.T) {
	LoadEmacsLocationHistory(emacsLocationHistoryFilename)
}

func TestNeoVimMsgPack(t *testing.T) {
	curdir, err := os.Getwd()
	if err != nil {
		t.Fail()
	}
	searchFilename, err := filepath.Abs(filepath.Join(curdir, "main.go"))
	if err != nil {
		t.Fail()
	}
	line, err := FindInNvimLocationHistory(nvimLocationHistoryFilename, searchFilename)
	if err != nil {
		// main.go might not be in the neovim location history, this is fine
		fmt.Println(err)
	}
	_ = line

	// Enable this for debugging
	// fmt.Println("line", line)
}
