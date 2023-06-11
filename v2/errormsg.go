package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

func quitError(tty *vt100.TTY, err error) {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	textoutput.NewTextOutput(true, true).Err(err.Error())
	vt100.SetXY(uint(0), uint(1))
	quitMut.Lock()
	defer quitMut.Unlock()
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
	quitMut.Lock()
	defer quitMut.Unlock()
	os.Exit(1)
}

func quitMessageWithStack(tty *vt100.TTY, msg string) {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	vt100.SetXY(uint(0), uint(newLineCount+1))
	debug.PrintStack()
	quitMut.Lock()
	defer quitMut.Unlock()
	os.Exit(1)
}
