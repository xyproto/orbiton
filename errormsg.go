package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xyproto/vt100"
)

func quitError(tty *vt100.TTY, err error) {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	fmt.Fprintln(os.Stderr, "error: "+err.Error())
	vt100.SetXY(uint(0), uint(1))
	os.Exit(1)
}

func quitMessage(tty *vt100.TTY, msg string) {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	vt100.SetXY(uint(0), uint(newLineCount+1))
	os.Exit(1)
}
