package main

import (
	"path/filepath"
	"strings"

	"github.com/xyproto/vt"
)

// handleDebugKey handles keypresses when in debug mode.
// Returns true if the key was consumed, false if it should be handled normally.
func (e *Editor) handleDebugKey(key string, c *vt.Canvas, tty *vt.TTY, status *StatusBar, debugEscCounter *int, undo *Undo) bool {
	switch key {
	case "c:17": // ctrl-q, exit debug mode
		e.debugMode = false
		e.DebugEnd()
		e.redraw.Store(true)
		status.SetMessageAfterRedraw("Default mode")
		return true

	case "c:23": // ctrl-w, add watch
		if expression, ok := e.UserInput(c, tty, status, "Variable name to watch", "", []string{}, false, ""); ok {
			if e.debugger == nil {
				e.debugger = newGDBDebugger(e.mode)
			}
			if _, err := e.debugger.AddWatch(expression); err != nil {
				status.ClearAll(c, true)
				status.SetError(err)
				status.ShowNoTimeout(c, e)
				return true
			}
		}
		return true

	case "c:6": // ctrl-f, step out
		if e.debugger == nil {
			// Auto-start the debug session
			if err := e.DebugStartSession(c, tty, status, ""); err != nil {
				status.ClearAll(c, false)
				status.SetError(err)
				status.ShowNoTimeout(c, e)
				e.redrawCursor.Store(true)
			} else {
				e.redrawCursor.Store(true)
				status.SetMessageAfterRedraw(status.Message())
			}
			return true
		}
		if e.debugComplete.Load() {
			e.DebugEnd()
			status.SetMessage("Execution complete")
			e.redrawCursor.Store(true)
			status.SetMessageAfterRedraw(status.Message())
			return true
		}
		status.ClearAll(c, false)
		if err := e.debugger.Finish(); err != nil {
			if err == errRecordingStopped {
				status.SetMessage("Recording stopped, press again to continue")
			} else if err == errProgramStopped || strings.Contains(err.Error(), "finish") {
				e.DebugEnd()
				status.SetMessage("Execution complete")
			} else {
				e.DebugEnd()
				status.SetMessage(err.Error())
			}
			e.GoToEnd(c, nil)
		} else if e.debugComplete.Load() {
			e.DebugEnd()
			status.SetMessage("Execution complete")
		} else {
			status.SetMessage("Step out")
		}
		e.redrawCursor.Store(true)
		status.SetMessageAfterRedraw(status.Message())
		return true

	case "c:0": // ctrl-space, continue
		if e.debugger != nil {
			if e.debugComplete.Load() {
				e.DebugEnd()
				status.SetMessage("Execution complete")
				e.redrawCursor.Store(true)
				status.SetMessageAfterRedraw(status.Message())
				return true
			}
			// Continue running to the next breakpoint or end
			if err := e.debugger.Continue(); err != nil {
				e.DebugEnd()
				status.SetMessage(err.Error())
				e.GoToEnd(c, nil)
			} else if e.debugComplete.Load() {
				e.DebugEnd()
				status.SetMessage("Execution complete")
			} else {
				status.SetMessage("Continue")
			}
			e.redrawCursor.Store(true)
			status.SetMessageAfterRedraw(status.Message())
		} else {
			// Auto-start the debug session
			if err := e.DebugStartSession(c, tty, status, ""); err != nil {
				status.ClearAll(c, false)
				status.SetError(err)
				status.ShowNoTimeout(c, e)
				e.redrawCursor.Store(true)
			} else {
				e.redrawCursor.Store(true)
				status.SetMessageAfterRedraw(status.Message())
			}
		}
		return true

	case "c:15": // ctrl-o, step over
		if e.debugger == nil {
			// Auto-start the debug session
			if err := e.DebugStartSession(c, tty, status, ""); err != nil {
				status.ClearAll(c, false)
				status.SetError(err)
				status.ShowNoTimeout(c, e)
				e.redrawCursor.Store(true)
			} else {
				e.redrawCursor.Store(true)
				status.SetMessageAfterRedraw(status.Message())
			}
			return true
		}
		if e.debugComplete.Load() {
			e.DebugEnd()
			status.SetMessage("Execution complete")
			e.redrawCursor.Store(true)
			status.SetMessageAfterRedraw(status.Message())
			return true
		}
		status.ClearAll(c, false)
		e.debugLastStepWasInstruction = false
		if err := e.debugger.Next(); err != nil {
			if errorMessage := err.Error(); strings.Contains(errorMessage, "is not being run") {
				e.DebugEnd()
				status.SetMessage("Done stepping")
			} else if err == errProgramStopped {
				e.DebugEnd()
				status.SetMessage("Execution complete")
			} else if err == errRecordingStopped {
				status.SetMessage("Recording stopped, press again to continue")
			} else {
				e.DebugEnd()
				status.SetMessage(errorMessage)
			}
			e.GoToEnd(c, nil)
		} else if e.debugComplete.Load() {
			e.DebugEnd()
			status.SetMessage("Execution complete")
		} else {
			status.SetMessage("Step over")
		}
		e.redrawCursor.Store(true)
		status.SetMessageAfterRedraw(status.Message())
		return true

	case "c:16": // ctrl-p, cycle register pane layout
		// e.showRegisters has three states, 0 (SmallRegisterWindow), 1 (LargeRegisterWindow) and 2 (NoRegisterWindow)
		e.debugShowRegisters++
		if e.debugShowRegisters > noRegisterWindow {
			e.debugShowRegisters = smallRegisterWindow
		}
		return true

	case "c:14": // ctrl-n, next instruction
		if e.debugger != nil {
			if e.debugComplete.Load() {
				e.DebugEnd()
				status.SetMessage("Execution complete")
				status.SetMessageAfterRedraw(status.Message())
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				return true
			}
			if !e.debugger.ProgramRunning() {
				e.DebugEnd()
				status.SetMessage("Execution complete")
				status.SetMessageAfterRedraw(status.Message())
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				return true
			}
			e.debugLastStepWasInstruction = true
			if err := e.debugger.NextInstruction(); err != nil {
				if errorMessage := err.Error(); strings.Contains(errorMessage, "is not being run") {
					e.DebugEnd()
					status.SetMessage("Could not start GDB")
				} else if err == errProgramStopped {
					e.DebugEnd()
					status.SetMessage("Execution complete")
				} else if err == errRecordingStopped {
					status.SetMessage("Recording stopped, press again to continue")
				} else { // got an unrecognized error
					e.DebugEnd()
					status.SetMessage(errorMessage)
				}
			} else {
				if e.debugComplete.Load() {
					e.DebugEnd()
					status.SetMessage("Execution complete")
				} else if !e.debugger.ProgramRunning() {
					e.DebugEnd()
					status.SetMessage("Program stopped when stepping") // Next instruction
				} else {
					// Don't show a status message per instruction/step when pressing ctrl-n
					return true
				}
			}
			e.redrawCursor.Store(true)
			status.SetMessageAfterRedraw(status.Message())
			return true
		}
		// e.debugger == nil: Build or export the current file
		outputExecutable, err := e.BuildOrExport(tty, c, status)
		// All clear when it comes to status messages and redrawing
		status.ClearAll(c, false)
		if err != nil && err != errNoSuitableBuildCommand {
			// Error while building
			status.SetError(err)
			status.ShowNoTimeout(c, e)
			e.debugMode = false
			e.redrawCursor.Store(true)
			e.redraw.Store(true)
			return true
		}
		// Was no suitable compilation or export command found?
		if err == errNoSuitableBuildCommand {
			// Both in debug mode and can not find a command to build this file with.
			status.SetError(err)
			status.ShowNoTimeout(c, e)
			e.debugMode = false
			e.redrawCursor.Store(true)
			e.redraw.Store(true)
			return true
		}
		// Start debugging
		if err := e.DebugStartSession(c, tty, status, outputExecutable); err != nil {
			status.ClearAll(c, false)
			status.SetError(err)
			status.ShowNoTimeout(c, e)
			e.redrawCursor.Store(true)
		}
		return true

	case "c:27": // esc, increment counter and toggle keybindings at >=3
		*debugEscCounter++
		if *debugEscCounter >= 3 {
			*debugEscCounter = 0
			e.debugHideKeybindings = !e.debugHideKeybindings
			e.redraw.Store(true)
		}
		return true

	case "c:9": // ctrl-i/tab, step into
		if e.debugger == nil {
			// Auto-start the debug session
			if err := e.DebugStartSession(c, tty, status, ""); err != nil {
				status.ClearAll(c, false)
				status.SetError(err)
				status.ShowNoTimeout(c, e)
				e.redrawCursor.Store(true)
			} else {
				e.redrawCursor.Store(true)
				status.SetMessageAfterRedraw(status.Message())
			}
			return true
		}
		if e.debugComplete.Load() {
			e.DebugEnd()
			status.SetMessage("Execution complete")
			e.redrawCursor.Store(true)
			status.SetMessageAfterRedraw(status.Message())
			return true
		}
		status.ClearAll(c, false)
		e.debugLastStepWasInstruction = false
		if err := e.debugger.Step(); err != nil {
			if errorMessage := err.Error(); strings.Contains(errorMessage, "is not being run") {
				e.DebugEnd()
				status.SetMessage("Done stepping")
			} else if err == errProgramStopped {
				e.DebugEnd()
				status.SetMessage("Execution complete")
			} else if err == errRecordingStopped {
				status.SetMessage("Recording stopped, press again to continue")
			} else {
				e.DebugEnd()
				status.SetMessage(errorMessage)
			}
			e.GoToEnd(c, nil)
		} else if e.debugComplete.Load() {
			e.DebugEnd()
			status.SetMessage("Execution complete")
		} else {
			status.SetMessage("Step into")
		}
		e.redrawCursor.Store(true)
		status.SetMessageAfterRedraw(status.Message())
		return true

	case "c:19": // ctrl-s, toggle stdout output
		e.debugHideOutput = !e.debugHideOutput
		e.redraw.Store(true)
		return true

	case "c:7": // ctrl-g, toggle GDB console
		e.debugShowConsole = !e.debugShowConsole
		e.redraw.Store(true)
		return true

	case "c:11": // ctrl-k, toggle keybindings
		e.debugHideKeybindings = !e.debugHideKeybindings
		e.redraw.Store(true)
		return true

	case "c:18": // ctrl-r, reverse step
		if e.debugger == nil {
			return false
		}
		if e.debugComplete.Load() {
			e.DebugEnd()
			status.SetMessage("Execution complete")
			e.redrawCursor.Store(true)
			status.SetMessageAfterRedraw(status.Message())
			return true
		}
		status.ClearAll(c, false)
		var err error
		var msg string
		if e.debugLastStepWasInstruction {
			err = e.debugger.ReverseNextInstruction()
			msg = "Reverse instruction"
		} else {
			err = e.debugger.ReverseStep()
			msg = "Reverse step"
		}
		if err != nil {
			if err == errProgramStopped {
				e.DebugEnd()
				status.SetMessage("Execution complete")
			} else if err == errRecordingStopped {
				status.SetMessage("Recording stopped")
			} else {
				status.SetMessage(err.Error())
			}
		} else if e.debugComplete.Load() {
			e.DebugEnd()
			status.SetMessage("Execution complete")
		} else {
			status.SetMessage(msg)
		}
		e.redrawCursor.Store(true)
		status.SetMessageAfterRedraw(status.Message())
		return true

	case "c:2": // ctrl-b, toggle breakpoint
		status.ClearAll(c, false)
		if e.breakpoint == nil {
			e.breakpoint = e.pos.Copy()
			_, err := e.DebugActivateBreakpoint(filepath.Base(e.filename))
			if err != nil {
				status.SetError(err)
				status.Show(c, e)
				e.redrawCursor.Store(true)
				return true
			}
			s := "Placed breakpoint at line " + e.LineNumber().String()
			status.SetMessage("  " + s + "  ")
		} else if e.breakpoint.LineNumber() == e.LineNumber() {
			// setting a breakpoint at the same line twice: remove the breakpoint
			s := "Removed breakpoint at line " + e.breakpoint.LineNumber().String()
			status.SetMessage(s)
			e.breakpoint = nil
		} else {
			undo.Snapshot(e)
			// Go to the breakpoint position
			e.GoToPosition(c, status, *e.breakpoint)
			// Do the redraw manually before showing the status message
			e.HideCursorDrawLines(c, true, false, true)
			e.redraw.Store(false)
			// Show the status message
			s := "Jumped to breakpoint at line " + e.LineNumber().String()
			status.SetMessage(s)
		}
		status.Show(c, e)
		e.redrawCursor.Store(true)
		return true

	case "c:3": // ctrl-c, clear watches
		if e.debugger != nil {
			wm := e.debugger.WatchMap()
			for k := range wm {
				delete(wm, k)
			}
		}
		e.debugWatches = nil
		e.redraw.Store(true)
		status.SetMessage("Cleared watches")
		status.SetMessageAfterRedraw(status.Message())
		return true
	}

	return false
}
