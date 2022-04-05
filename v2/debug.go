package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/cyrus-and/gdb"
	"github.com/xyproto/mode"
)

var (
	gdbOutput         bytes.Buffer
	originalDirectory string
)

// DebugStart will start a new debug session, using gdb.
// Will end the existing session first if e.gdb != nil.
func (e *Editor) DebugStart(directory, sourceBaseFilename, executableBaseFilename string) (string, error) {

	//logf("debug: dir %s, src %s, exe %s\n", directory, sourceBaseFilename, executableBaseFilename)

	// End any existing sessions
	e.DebugEnd()

	// Change directory to the sourcefile, temporarily
	var err error
	originalDirectory, err = os.Getwd()
	if err == nil { // cd success
		err = os.Chdir(directory)
		if err != nil {
			return "", errors.New("could not change directory to " + directory)
		}
	}

	// Use rust-gdb if we are debugging Rust
	var gdbExecutable string
	if e.mode == mode.Rust {
		gdbExecutable = which("rust-db")
	} else {
		gdbExecutable = which("gdb")
	}

	// Start a new gdb session
	e.gdb, err = gdb.NewCustom(gdbExecutable, func(notification map[string]interface{}) {
		// Handle messages from gdb, including frames that contains line numbers
		if payload, ok := notification["payload"]; ok && notification["type"] == "exec" {
			if payloadMap, ok := payload.(map[string]interface{}); ok {
				if frame, ok := payloadMap["frame"]; ok {
					if frameMap, ok := frame.(map[string]interface{}); ok {
						if lineNumberString, ok := frameMap["line"].(string); ok {
							if lineNumber, err := strconv.Atoi(lineNumberString); err == nil { // success
								// Got a line number, send the editor there, without any status messages
								e.GoToLineNumber(LineNumber(lineNumber), nil, nil, true)
							}
						}
					}
				}
			}
		}
	})

	if err != nil {
		e.gdb = nil
		return "", err
	}
	if e.gdb == nil {
		return "", errors.New("gdb.New returned no error, but e.gdb is nil")
	}

	// Handle output to stdout (and stderr?) from programs that are being debugged
	go io.Copy(&gdbOutput, e.gdb)

	// Load the executable file
	if retvalMap, err := e.gdb.CheckedSend("file-exec-and-symbols", executableBaseFilename); err != nil {
		return fmt.Sprintf("%v", retvalMap), err
	}

	// Pass in arguments
	//e.gdb.Send("exec-arguments", "--version")

	// Pass the breakpoint, if it has been set with ctrl-b
	if e.breakpoint != nil {
		if retvalMap, err := e.gdb.CheckedSend("break-insert", fmt.Sprintf("%s:%d", sourceBaseFilename, e.breakpoint.LineNumber())); err != nil {
			return fmt.Sprintf("%v", retvalMap), err
		}
	}

	// Start from the top
	if _, err := e.gdb.CheckedSend("exec-run", "--start"); err != nil {
		return gdbOutput.String(), err
	}

	return "started gdb", nil
}

// DebugContinue will continue the execution to the next breakpoint or to the end.
// e.gdb must not be nil. Returns whatever was outputted to gdb stdout.
func (e *Editor) DebugContinue() (string, error) {
	_, err := e.gdb.CheckedSend("exec-continue")
	if err != nil {
		return "", err
	}
	output := gdbOutput.String()
	gdbOutput.Reset()
	return output, nil
}

// DebugStep will continue the execution by stepping to the next line.
// e.gdb must not be nil. Returns whatever was outputted to gdb stdout.
func (e *Editor) DebugStep() (string, error) {
	//logf("%s\n", "[step] start")
	//logf("%s\n", "[step] sending exec-step")
	_, err := e.gdb.CheckedSend("exec-step")
	if err != nil {
		return "", err
	}
	//logf("[step] got %v and no error\n", retval)
	output := gdbOutput.String()
	//logf("[step] got GDB output: %s\n", output)
	gdbOutput.Reset()
	//logf("[step] end, returning output: %s\n", output)
	return output, nil
}

// DebugEnd will end the current gdb session
func (e *Editor) DebugEnd() {
	if e.gdb != nil {
		e.gdb.Exit()
	}
	e.gdb = nil
	// Clear any existing output
	gdbOutput.Reset()
	// Also change to the original directory
	if originalDirectory != "" {
		os.Chdir(originalDirectory)
	}
}
