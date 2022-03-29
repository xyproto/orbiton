package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cyrus-and/gdb"
)

var gdbOutput bytes.Buffer

// DebugStart will start a new debug session, using gdb.
// Will end the existing session first if e.gdb != nil.
func (e *Editor) DebugStart(sourceFilename, executableFilename string) (string, error) {
	// End any existing sessions
	e.DebugEnd()

	// Change directory to the sourcefile, temporarily
	curDir, err := os.Getwd()
	if err == nil { // cd success
		err = os.Chdir(filepath.Dir(sourceFilename))
		if err != nil {
			return "", errors.New("could not change directory to " + filepath.Dir(sourceFilename))
		}
		defer os.Chdir(curDir)
	}

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

	// Load the executable file
	if retvalMap, err := e.gdb.CheckedSend("file-exec-and-symbols", executableFilename); err != nil {
		return fmt.Sprintf("%v", retvalMap), err
	}

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
		//logf("could not exec-run: %s\n", err)
		return gdbOutput.String(), err
	}

	//logf("%s\n", "could exec-run")
	//logf("gdb stdout: %s\n", gdbOutput.String())

	return "started gdb", nil
}

// DebugContinue will continue the execution to the next breakpoint or to the end.
// e.gdb must not be nil. Returns whatever was outputted to gdb stdout.
func (e *Editor) DebugContinue() (string, error) {
	_, err := e.gdb.CheckedSend("exec-continue")
	if err != nil {
		return "", err
	}
	return gdbOutput.String(), nil
}

// DebugStep will continue the execution by stepping to the next line.
// e.gdb must not be nil. Returns whatever was outputted to gdb stdout.
func (e *Editor) DebugStep() (string, error) {
	_, err := e.gdb.CheckedSend("exec-step")
	if err != nil {
		return "", err
	}
	return gdbOutput.String(), nil
}

// DebugEnd will end the current gdb session
func (e *Editor) DebugEnd() {
	if e.gdb != nil {
		e.gdb.Exit()
	}
	e.gdb = nil
	// Clear any existing output
	gdbOutput.Reset()
}
