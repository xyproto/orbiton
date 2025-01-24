package main

import (
	"os"
	"testing"

	"github.com/xyproto/vt100"
)

func TestMain(m *testing.M) {
	code := m.Run()

	//vt100.Close()
	//vt100.Reset()
	vt100.ShowCursor(true)

	os.Exit(code)
}
