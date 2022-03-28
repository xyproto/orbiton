package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cyrus-and/gdb"
	"github.com/xyproto/vt100"
)

func (e *Editor) StartGDB(c *vt100.Canvas, status *StatusBar, absFilename, outputExecutable string) {
	var err error
	// Try to start a new gdb session
	if e.gdb == nil {
		e.gdb, err = gdb.New(func(notification map[string]interface{}) {
			notificationText, err := json.Marshal(notification)
			if err != nil {
				log.Fatal(err)
			}
			logf("%s\n", notificationText)
		})
		if err != nil {
			status.ClearAll(c)
			status.SetErrorMessage(err.Error())
			status.Show(c, e)
			return
		}
		if e.gdb == nil {
			status.ClearAll(c)
			status.SetErrorMessage("could not start gdb")
			status.Show(c, e)
			return
		}
	}

	//defer e.gdb.Exit()

	// Load the executable file
	e.gdb.Send("file-exec-file", outputExecutable)
	// Pass in arguments
	//e.gdb.Send("exec-arguments", "--version")
	// Pass the breakpoint, if it has been set with ctrl-b
	if e.breakpoint != nil {
		e.gdb.Send("break-insert", fmt.Sprintf("%s:%d", absFilename, e.breakpoint.LineNumber()))
	}

	go func() {
		// Start from the top
		e.gdb.Send("exec-run", "--start")
	}()

	//logf("%s\n", "not dead")
	//e.gdb.Exit()
	//e.gdb = nil
}
