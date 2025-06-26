package main

// The whole purpose of this file is to make sure the cursor is not hidden after all the tests have been run.

import (
	"os"
	"testing"

	"github.com/xyproto/vt"
)

func TestMain(m *testing.M) {
	code := m.Run()

	vt.ShowCursor(true)

	os.Exit(code)
}
