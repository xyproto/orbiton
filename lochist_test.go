package main

import (
	"testing"
)

func TestVimInfo(t *testing.T) {
	LoadVimLocationHistory(expandUser(vimLocationHistoryFilename))
}

func TestEmacsPlaces(t *testing.T) {
	LoadEmacsLocationHistory(expandUser(emacsLocationHistoryFilename))
}
