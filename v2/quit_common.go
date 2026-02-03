package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// quitError stops the program with an error message
func quitError(tty *vt.TTY, err error) {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt.Reset()
	vt.SetNoColor()
	vt.Clear()
	vt.Close()
	vt.New().Err(err.Error())
	vt.ShowCursor(true)
	vt.SetXY(uint(0), uint(1))
	os.Exit(1)
}

// quitMessage stops the program with a message
func quitMessage(tty *vt.TTY, msg string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt.Reset()
	vt.SetNoColor()
	vt.Clear()
	vt.Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	vt.ShowCursor(true)
	vt.SetXY(uint(0), uint(newLineCount+1))
	os.Exit(1)
}

// quitMessageWithStack stops the program with a message and stack trace
func quitMessageWithStack(tty *vt.TTY, msg string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt.Reset()
	vt.SetNoColor()
	vt.Clear()
	vt.Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	vt.ShowCursor(true)
	vt.SetXY(uint(0), uint(newLineCount+1))
	debug.PrintStack()
	os.Exit(1)
}

// CatBytes detects the source code mode and outputs syntax highlighted text to the given TextOutput.
func CatBytes(sourceCodeData []byte, o *vt.TextOutput) error {
	detectedMode := mode.SimpleDetectBytes(sourceCodeData)
	taggedTextBytes, err := AsText(sourceCodeData, detectedMode)
	if err == nil {
		o.OutputTags(string(taggedTextBytes))
	}
	return err
}

// quitCat outputs syntax highlighted source code and exits
func quitCat(fnord *FilenameOrData) {
	quitMut.Lock()
	defer quitMut.Unlock()
	if fnord.Empty() {
		if sourceCodeBytes, err := os.ReadFile(fnord.filename); err == nil { // success
			if err := CatBytes(sourceCodeBytes, tout); err == nil { // success
				vt.ShowCursor(true)
				os.Exit(0)
			}
		}
	} else {
		if err := CatBytes(fnord.data, tout); err == nil { // success
			vt.ShowCursor(true)
			os.Exit(0)
		}
	}
	vt.ShowCursor(true)
	os.Exit(1) // could not cat the file in a syntax highlighted way
}
