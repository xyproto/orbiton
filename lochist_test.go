package main

import (
	"testing"
)

func TestViminfo(t *testing.T) {
	LoadVimLocationHistory(expandUser(vimLocationHistoryFilename))
}
