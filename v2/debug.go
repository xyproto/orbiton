package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cyrus-and/gdb"
	"github.com/xyproto/vt100"
)

var gdbOutput bytes.Buffer

// DebugStart will start a new debug session, using gdb.
// Will end the existing session first if e.gdb != nil.
func (e *Editor) DebugStart(c *vt100.Canvas, status *StatusBar, sourceFilename, executableFilename string) (string, error) {

	// End any existing sessions
	if e.gdb != nil {
		e.gdb.Exit()
		e.gdb = nil
	}

	// Change directory to the sourcefile, temporarily
	curDir, err := os.Getwd()
	if err == nil { // cd success
		err = os.Chdir(filepath.Dir(sourceFilename))
		if err != nil {
			return "", errors.New("could not change directory to " + filepath.Dir(sourceFilename))
		}
		defer os.Chdir(curDir)
	}

	cd2, _ := os.Getwd()
	logf("CURRENT DIRECTORY: %s\n", cd2)

	// Start a new gdb session
	e.gdb, err = gdb.New(nil)
	go io.Copy(&gdbOutput, e.gdb)
	if err != nil {
		e.gdb = nil
		return "", err
	}
	if e.gdb == nil {
		return "", errors.New("could not start gdb even though err == nil")
	}

	// 	// Set the source directory
	// 	sourceDir := filepath.Dir(sourceFilename)
	// 	if retvalMap, err := e.gdb.CheckedSend("dir", sourceDir); err != nil {
	// 		return fmt.Sprintf("%v", retvalMap), err
	// 	}

	// Load the executable file
	if retvalMap, err := e.gdb.CheckedSend("file-exec-and-symbols", executableFilename); err != nil {
		return fmt.Sprintf("%v", retvalMap), err
	}

	// 	// Load the source file
	// 	if retvalMap, err := e.gdb.CheckedSend("file", filepath.Base(sourceFilename)); err != nil {
	// 		return fmt.Sprintf("%v", retvalMap), err
	// 	}

	// Pass in arguments
	//e.gdb.Send("exec-arguments", "--version")

	// Pass the breakpoint, if it has been set with ctrl-b
	if e.breakpoint != nil {
		if retvalMap, err := e.gdb.CheckedSend("break-insert", fmt.Sprintf("%s:%d", sourceFilename, e.breakpoint.LineNumber())); err != nil {
			return fmt.Sprintf("%v", retvalMap), err
		}
	}

	// Start from the top, in a goroutine
	//go func() {
	if _, err := e.gdb.CheckedSend("exec-run", "--start"); err != nil {
		status.SetMessage("Could not exec-run with gdb")
		status.Show(c, e)
		logf("could not exec-run: %s\n", err)
		logf("gdb stdout: %s\n", gdbOutput.String())
		return gdbOutput.String(), err
	}
	status.SetMessage("Could exec-run with gdb")
	status.Show(c, e)
	//logf("gdb stdout: %s\n", gdbOutput.String())
	//}()

	return "started gdb", nil
}

// DebugContinue will continue the execution to the next breakpoint or to the end.
// e.gdb must not be nil.
func (e *Editor) DebugContinue() (string, error) {
	retvalMap, err := e.gdb.CheckedSend("exec-continue")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", retvalMap), nil
}

// DebugStep will continue the execution by stepping to the next line.
// e.gdb must not be nil.
func (e *Editor) DebugStep() (string, error) {
	retvalMap, err := e.gdb.CheckedSend("exec-step")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", retvalMap), nil
}

// DebugFrame will return the current gdb frame as a string
func (e *Editor) DebugFrame() (string, error) {
	retvalMap, err := e.gdb.CheckedSend("frame")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", retvalMap), nil
}
