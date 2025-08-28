// Package main implements the bubbletea integration for Orbiton editor
package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/mode"
)

// LoopBubbletea runs the main editor loop using bubbletea
// This is the new bubbletea-based replacement for the original Loop function
func LoopBubbletea(tty *TTY, fnord FilenameOrData, lineNumber LineNumber, colNumber ColNumber, forceFlag bool, theme Theme, syntaxHighlight, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp, fmtFlag bool) (userMessage string, stopParent bool, err error) {

	// Initialize the VT100 terminal
	Init()

	// Create the bubbletea model
	model, err := NewOrbitonModel(tty, fnord, lineNumber, colNumber, forceFlag, theme, syntaxHighlight, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp, fmtFlag)
	if err != nil {
		clearOnQuit.Store(false)
		return "", false, err
	}

	// Get absolute filename for locking
	absFilename := fnord.filename
	if !fnord.stdin && model.editor != nil {
		if filename, err := model.editor.AbsFilename(); err == nil {
			absFilename = filename
		}
	}

	// Handle parent process detection (man page mode)
	if parentIsMan == nil {
		b := parentProcessIs("man")
		parentIsMan = &b
	}
	if *parentIsMan && model.editor != nil {
		if env.Has("ORBITON_SPACE") && env.No("NROFF_FILENAME") {
			model.editor.mode = mode.Nroff
		} else {
			model.editor.mode = mode.ManPage
		}
	}

	// Minor adjustments to some modes
	if model.editor != nil {
		switch model.editor.mode {
		case mode.Email, mode.Git:
			model.editor.StatusForeground = LightBlue
			model.editor.StatusBackground = BackgroundDefault
		case mode.ManPage:
			model.editor.readOnly = true
		}
	}

	// Set up signal handlers
	if model.editor != nil {
		const onlyClearSignals = false
		model.editor.SetUpSignalHandlers(model.canvas, tty, model.status, onlyClearSignals)
	}

	// Monitor read-only file if requested
	if monitorAndReadOnly && model.editor != nil {
		model.editor.readOnly = true
		if err := model.editor.StartMonitoring(model.canvas, tty, model.status); err != nil {
			quitError(tty, err)
		}
	}

	if model.editor != nil {
		if model.editor.mode == mode.Log && model.editor.readOnly {
			model.editor.syntaxHighlight = true
		}

		model.editor.previousX = 1
		model.editor.previousY = 1
	}

	// Set TTY timeout
	tty.SetTimeout(2 * time.Millisecond)

	// Handle file locking
	var lockTimestamp time.Time
	canUseLocks.Store(!fnord.stdin && !monitorAndReadOnly)

	if canUseLocks.Load() {
		go func() {
			if err := fileLock.Load(); err != nil {
				if err := fileLock.Save(); err != nil {
					canUseLocks.Store(false)
				}
			}
		}()

		// Check if lock should be forced
		if forceFlag || filepath.Base(absFilename) == "COMMIT_EDITMSG" || env.Bool("O_FORCE") {
			go func() {
				fileLock.Lock(absFilename)
				fileLock.Save()
			}()
		} else {
			if err := fileLock.Lock(absFilename); err != nil {
				return fmt.Sprintf("Locked by another (possibly dead) instance of this editor.\nTry: o -f %s", filepath.Base(absFilename)), false, errors.New(absFilename + " is locked")
			}
			go fileLock.Save()
		}
		lockTimestamp = fileLock.GetTimestamp(absFilename)

		// Set up panic recovery for unlocking
		defer func() {
			if x := recover(); x != nil {
				go func() {
					quitMut.Lock()
					defer quitMut.Unlock()
					fileLock.Unlock(absFilename)
					fileLock.Save()
				}()

				msg := fmt.Sprintf("Saved the file first!\n%v", x)
				if model.editor != nil {
					if err := model.editor.Save(model.canvas, tty); err != nil {
						msg = fmt.Sprintf("Could not save the file first! %v\n%v", err, x)
					}
				}
				quitMessageWithStack(tty, msg)
			}
		}()
	}

	// Handle format-only flag
	if fmtFlag && model.editor != nil {
		if model.editor.mode == mode.Markdown {
			model.editor.GoToStartOfTextLine(model.canvas)
			model.editor.FormatAllMarkdownTables()
		}
		model.editor.formatCode(model.canvas, tty, model.status, &model.jsonFormatToggle)
		if msg := strings.TrimSpace(model.status.msg); model.status.isError && msg != "" {
			quitError(tty, errors.New(msg))
		}
		model.editor.UserSave(model.canvas, tty, model.status)
		model.quit = true
	}

	// Initial redraw
	if model.editor != nil {
		model.editor.InitialRedraw(model.canvas, model.status)

		// Show quick help if enabled
		if (!QuickHelpScreenIsDisabled() || model.editor.displayQuickHelp) && !model.editor.noDisplayQuickHelp {
			model.editor.DrawQuickHelp(model.canvas, false)
		}

		// Place and enable cursor
		model.editor.PlaceAndEnableCursor()
	}

	// Create bubbletea program
	program := tea.NewProgram(
		*model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the bubbletea program
	finalModel, err := program.Run()
	if err != nil {
		return "", false, fmt.Errorf("bubbletea program error: %v", err)
	}

	// Extract final state
	finalOrbitonModel := finalModel.(OrbitonModel)

	// Clean up locks and location history
	var closeLocksWaitGroup sync.WaitGroup
	if finalOrbitonModel.editor != nil {
		finalOrbitonModel.editor.CloseLocksAndLocationHistory(absFilename, lockTimestamp, forceFlag, &closeLocksWaitGroup)
	}

	// Clear colors and terminal state
	SetNoColor()

	// Handle quit cleanup
	if clearOnQuit.Load() {
		Clear()
		Close()
	} else {
		if finalOrbitonModel.status != nil {
			finalOrbitonModel.status.ClearAll(finalOrbitonModel.canvas, false)
		}
		if finalOrbitonModel.canvas != nil {
			finalOrbitonModel.canvas.Draw()
		}
	}

	// Re-enable cursor
	ShowCursor(true)

	// Wait for cleanup
	closeLocksWaitGroup.Wait()

	// Stop background processes
	stopBackgroundProcesses()

	stopParentResult := false
	if finalOrbitonModel.editor != nil {
		stopParentResult = finalOrbitonModel.editor.stopParentOnQuit
	}

	return "", stopParentResult, nil
}

// shouldUseBubbletea determines whether to use bubbletea or the legacy loop
// This can be controlled by environment variables for gradual migration
func shouldUseBubbletea() bool {
	// Check for explicit override
	if env.Bool("ORBITON_USE_BUBBLETEA") {
		return true
	}
	if env.Bool("ORBITON_NO_BUBBLETEA") {
		return false
	}

	// Default to bubbletea for new installations
	// For now, we'll default to false to maintain compatibility
	return false
}

// RunLoop decides whether to use the new bubbletea loop or the legacy loop
func RunLoop(tty *TTY, fnord FilenameOrData, lineNumber LineNumber, colNumber ColNumber, forceFlag bool, theme Theme, syntaxHighlight, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp, fmtFlag bool) (userMessage string, stopParent bool, err error) {

	if shouldUseBubbletea() {
		return LoopBubbletea(tty, fnord, lineNumber, colNumber, forceFlag, theme, syntaxHighlight, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp, fmtFlag)
	}
	return Loop(tty, fnord, lineNumber, colNumber, forceFlag, theme, syntaxHighlight, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp, fmtFlag)
}
