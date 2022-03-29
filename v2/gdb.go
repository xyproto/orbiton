package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cyrus-and/gdb"
	"github.com/xyproto/vt100"
)

// DebugStart will start a new debug session, using gdb.
// Will end the existing session first if e.gdb != nil.
func (e *Editor) DebugStart(c *vt100.Canvas, status *StatusBar, absFilename, outputExecutable string) (string, error) {

	// End any existing sessions
	if e.gdb != nil {
		e.gdb.Exit()
		e.gdb = nil
	}

	// Start a new gdb session
	var err error
	var retvalJSON []byte
	e.gdb, err = gdb.New(func(notification map[string]interface{}) {
		s, _ := json.Marshal(notification)
		logf("starting gdb, got %s and %s\n", s, err.Error())
	})
	if err != nil {
		e.gdb = nil
		return string(retvalJSON), err
	}
	if e.gdb == nil {
		return "", errors.New("could not start gdb")
	}

	// Load the executable file
	if retvalMap, err := e.gdb.CheckedSend("file-exec-file", outputExecutable); err != nil {
		return fmt.Sprintf("%v", retvalMap), err
	}

	// Pass in arguments
	//e.gdb.Send("exec-arguments", "--version")

	// Pass the breakpoint, if it has been set with ctrl-b
	if e.breakpoint != nil {
		if retvalMap, err := e.gdb.CheckedSend("break-insert", fmt.Sprintf("%s:%d", absFilename, e.breakpoint.LineNumber())); err != nil {
			return fmt.Sprintf("%v", retvalMap), err
		}
	}

	// Start from the top, in a goroutine
	go func() {
		if _, err := e.gdb.CheckedSend("exec-run", "--start"); err != nil {
			status.SetMessage("Could not exec-run with gdb")
			status.Show(c, e)
		} else {
			status.SetMessage("Could exec-run with gdb")
			status.Show(c, e)
		}
	}()

	return "started gdb", nil
}

// DebugContinue will continue the exeuction by stepping to the next line.
// e.gdb must not be nil.
func (e *Editor) DebugContinue() (string, error) {
	retvalMap, err := e.gdb.CheckedSend("continue")
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
