package main

// The whole purpose of this file is to make sure the cursor is not hidden after all the tests have been run.

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()

	ShowCursor(true)

	os.Exit(code)
}
