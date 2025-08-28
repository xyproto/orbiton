// Package main implements the bubbletea model for Orbiton editor
package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xyproto/mode"
)

// OrbitonModel represents the state of the Orbiton editor using bubbletea
type OrbitonModel struct {
	// Core Orbiton components
	editor *Editor
	canvas *Canvas
	status *StatusBar
	tty    *TTY

	// Editor configuration
	fnord                      FilenameOrData
	lineNumber                 LineNumber
	colNumber                  ColNumber
	forceFlag                  bool
	theme                      Theme
	syntaxHighlight            bool
	monitorAndReadOnly         bool
	nanoMode                   bool
	createDirectoriesIfMissing bool
	displayQuickHelp           bool
	noDisplayQuickHelp         bool
	fmtFlag                    bool

	// Runtime state
	quit         bool
	redraw       bool
	redrawCursor bool

	// Key history and macro support
	keyHistory *KeyHistory
	bookmark   *Position

	// State tracking
	lastCopyY        LineIndex
	lastPasteY       LineIndex
	lastCutY         LineIndex
	firstPasteAction bool
	firstCopyAction  bool

	// Special modes and settings
	jsonFormatToggle       bool
	regularEditingRightNow bool

	// Arrow key held down tracking for speed up
	heldDownLeftArrowTime  time.Time
	heldDownRightArrowTime time.Time

	// Miscellaneous state
	clearKeyHistory bool
}

// Init initializes the bubbletea model
func (m OrbitonModel) Init() tea.Cmd {
	return nil
}

// Update handles incoming bubbletea messages and updates the model
func (m OrbitonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		// Handle terminal resize
		m.canvas.Resize()
		m.redraw = true
		return m, nil
	case tea.QuitMsg:
		m.quit = true
		return m, tea.Quit
	}

	return m, nil
}

// View renders the current state of the editor
func (m OrbitonModel) View() string {
	if m.redraw || m.redrawCursor {
		// Use the existing Canvas rendering system
		m.canvas.HideCursorAndDraw()

		// Handle status bar rendering
		if m.editor != nil && m.status != nil {
			if m.editor.nanoMode.Load() {
				m.status.Show(m.canvas, m.editor)
			} else if m.editor.statusMode {
				m.status.ShowFilenameLineColWordCount(m.canvas, m.editor)
			} else if m.editor.blockMode {
				m.status.ShowBlockModeStatusLine(m.canvas, m.editor)
			} else if m.status.IsError() {
				m.status.Show(m.canvas, m.editor)
			}
		}

		// Handle cursor positioning
		if m.redrawCursor && m.editor != nil {
			m.editor.EnableAndPlaceCursor(m.canvas)
		}
	}

	// Return empty string since we're using the existing VT100 rendering
	// The actual rendering happens through the Canvas system
	return ""
}

// handleKeyMsg processes keyboard input and maps it to Orbiton's key handling system
func (m OrbitonModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Convert bubbletea key message to Orbiton's key format
	key := m.convertKeyMsg(msg)

	// Process the key using Orbiton's existing key handling logic
	// This mirrors the switch statement from keyloop.go
	switch key {
	case "c:17": // ctrl-q, quit
		if m.editor != nil && m.editor.nanoMode.Load() { // nano: ctrl-w, search backwards
			const clearPreviousSearch = true
			const searchForward = false
			m.editor.SearchMode(m.canvas, m.status, m.tty, clearPreviousSearch, searchForward, undo)
			return m, nil
		}
		m.quit = true
		return m, tea.Quit

	case "c:19": // ctrl-s, save
		if m.editor != nil {
			if m.editor.debugMode {
				m.editor.DebugStep()
				return m, nil
			}
			m.editor.UserSave(m.canvas, m.tty, m.status)
		}
		return m, nil

	case "c:6": // ctrl-f, search
		if m.editor != nil {
			const clearPreviousSearch = true
			const searchForward = true
			m.editor.SearchMode(m.canvas, m.status, m.tty, clearPreviousSearch, searchForward, undo)
		}
		return m, nil

	case "c:7": // ctrl-g, go to definition or toggle status bar
		if m.editor != nil {
			if m.editor.nanoMode.Load() {
				// nano: ctrl-g, help
				m.status.ClearAll(m.canvas, false)
				const repositionCursorAfterDrawing = false
				m.editor.DrawNanoHelp(m.canvas, repositionCursorAfterDrawing)
				return m, nil
			}

			// Toggle status bar or go to definition
			if m.keyHistory.PrevIs("c:7") || len(m.editor.SearchTerm()) > 0 {
				m.editor.statusMode = !m.editor.statusMode
				m.redraw = true
			} else {
				jumpedToDefinition := m.editor.FuncPrefix() != "" && m.editor.GoToDefinition(m.tty, m.canvas, m.status)
				if !jumpedToDefinition {
					m.editor.statusMode = !m.editor.statusMode
					m.redraw = true
				}
			}
		}
		return m, nil

	case "c:15": // ctrl-o, command menu
		if m.editor != nil {
			m.status.ClearAll(m.canvas, false)
			undo.Snapshot(m.editor)
			const lastCommandMenuIndex = 0
			// TODO: We need access to forceFlag and fileLock from the model
			selectedIndex, spacePressed := m.editor.CommandMenu(m.canvas, m.tty, m.status, m.bookmark, undo, lastCommandMenuIndex, m.forceFlag, fileLock)
			_ = selectedIndex // TODO: Store this in model state
			if spacePressed {
				m.status.Clear(m.canvas, false)
				// Command execution would continue here
			}
			m.redraw = true
		}
		return m, nil

	case "c:0": // ctrl-space, build or export
		if m.editor != nil {
			if m.editor.nanoMode.Load() {
				// do nothing in nano mode
				return m, nil
			}
			m.editor.runAfterBuild = m.keyHistory.DoubleTapped("c:0")
			stopBackgroundProcesses()
			m.editor.Build(m.canvas, m.status, m.tty)
			m.redrawCursor = true
		}
		return m, nil

	case "c:20": // ctrl-t, toggle header/impl, record macro, etc.
		if m.editor != nil {
			// TODO: Implement full ctrl-t functionality
			// For now, just show a message
			m.status.SetMessage("ctrl-t functionality not yet implemented in bubbletea mode")
			m.status.Show(m.canvas, m.editor)
			m.redraw = true
		}
		return m, nil

	case "c:28": // ctrl-\, toggle comments
		if m.editor != nil {
			m.editor.ToggleCommentBlock(m.canvas)
		}
		return m, nil

	case "c:31": // ctrl-_, digraph or matching parenthesis
		if m.editor != nil {
			// Implementation from keyloop.go
			if m.editor.JumpToMatching(m.canvas) {
				m.redrawCursor = true
			} else {
				// TODO: Implement digraph insertion
				m.status.SetMessage("Digraph insertion not yet implemented in bubbletea mode")
				m.status.Show(m.canvas, m.editor)
			}
			m.redraw = true
		}
		return m, nil

	case "c:16": // ctrl-p, scroll up or previous match
		if m.editor != nil {
			if len(m.editor.SearchTerm()) > 0 {
				const searchBackwards = true
				const searchForwards = false
				m.editor.GoToNextMatch(m.canvas, m.status, searchBackwards, searchForwards)
			} else {
				m.redraw = m.editor.ScrollUp(m.canvas, m.status, 10)
			}
			m.redrawCursor = true
		}
		return m, nil

	case "c:14": // ctrl-n, scroll down or next match
		if m.editor != nil {
			if len(m.editor.SearchTerm()) > 0 {
				const searchBackwards = false
				const searchForwards = true
				m.editor.GoToNextMatch(m.canvas, m.status, searchBackwards, searchForwards)
			} else {
				m.redraw = m.editor.ScrollDown(m.canvas, m.status, 10, int(m.canvas.H()))
			}
			m.redrawCursor = true
		}
		return m, nil

	case "c:12": // ctrl-l, go to line
		if m.editor != nil {
			// TODO: Fix method signature for GoToLineNumber
			m.status.SetMessage("Go to line not yet implemented in bubbletea mode")
			m.status.Show(m.canvas, m.editor)
			m.redraw = true
		}
		return m, nil

	case "c:25": // ctrl-y, scroll up one line
		if m.editor != nil {
			m.redraw = m.editor.ScrollUp(m.canvas, m.status, 1)
			m.redrawCursor = true
		}
		return m, nil

	case "c:4": // ctrl-d, delete character
		if m.editor != nil {
			undo.Snapshot(m.editor)
			m.editor.Delete(m.canvas, true)
			m.redraw = true
			m.redrawCursor = true
		}
		return m, nil

	case "c:21": // ctrl-u, undo
		if m.editor != nil {
			undo.Restore(m.editor)
			m.redraw = true
			m.redrawCursor = true
		}
		return m, nil

	case "c:26": // ctrl-z, undo
		if m.editor != nil {
			undo.Restore(m.editor)
			m.redraw = true
			m.redrawCursor = true
		}
		return m, nil

	case "c:24": // ctrl-x, cut line
		if m.editor != nil {
			// TODO: Implement cut functionality
			m.status.SetMessage("Cut not yet implemented in bubbletea mode")
			m.status.Show(m.canvas, m.editor)
			m.redraw = true
			m.redrawCursor = true
		}
		return m, nil

	case "c:11": // ctrl-k, delete to end of line
		if m.editor != nil {
			undo.Snapshot(m.editor)
			// TODO: Fix method signature for DeleteToEndOfLine
			m.status.SetMessage("Delete to end of line not yet implemented in bubbletea mode")
			m.status.Show(m.canvas, m.editor)
			m.redraw = true
			m.redrawCursor = true
		}
		return m, nil

	case "c:22": // ctrl-v, paste
		if m.editor != nil {
			// TODO: Implement paste functionality
			m.status.SetMessage("Paste not yet implemented in bubbletea mode")
			m.status.Show(m.canvas, m.editor)
			m.redraw = true
			m.redrawCursor = true
		}
		return m, nil

	case "c:18": // ctrl-r, portal
		if m.editor != nil {
			if m.editor.debugMode {
				// TODO: Implement debug continue
				m.status.SetMessage("Debug continue not yet implemented in bubbletea mode")
			} else {
				// TODO: Implement portal functionality
				m.status.SetMessage("Portal not yet implemented in bubbletea mode")
			}
			m.status.Show(m.canvas, m.editor)
		}
		return m, nil

	case "c:2": // ctrl-b, bookmark or go back
		if m.editor != nil {
			if m.editor.debugMode {
				// TODO: Implement breakpoint toggle
				m.status.SetMessage("Breakpoint toggle not yet implemented in bubbletea mode")
				m.status.Show(m.canvas, m.editor)
			} else {
				// TODO: Implement bookmark functionality properly
				m.status.SetMessage("Bookmark/go back not yet implemented in bubbletea mode")
				m.status.Show(m.canvas, m.editor)
			}
			m.redraw = true
			m.redrawCursor = true
		}
		return m, nil

	case "c:10": // ctrl-j, join lines
		if m.editor != nil {
			undo.Snapshot(m.editor)
			// TODO: Implement join lines functionality
			m.status.SetMessage("Join lines not yet implemented in bubbletea mode")
			m.status.Show(m.canvas, m.editor)
			m.redraw = true
			m.redrawCursor = true
		}
		return m, nil

	case "c:3": // ctrl-c, copy
		if m.editor != nil {
			// TODO: Implement copy functionality properly
			m.status.SetMessage("Copy not yet implemented in bubbletea mode")
			m.status.Show(m.canvas, m.editor)
		}
		return m, nil

	case "c:1", homeKey: // ctrl-a, home
		if m.editor != nil {
			// Simplified home functionality for now
			m.editor.Home()
			m.redrawCursor = true
		}
		return m, nil

	case "c:5", endKey: // ctrl-e, end
		if m.editor != nil {
			// Simplified end functionality for now
			m.editor.End(m.canvas)
			m.redrawCursor = true
		}
		return m, nil

	case "c:23": // ctrl-w, format or insert template
		if m.editor != nil && m.editor.nanoMode.Load() { // nano: ctrl-w, search
			const clearPreviousSearch = true
			const searchForward = true
			m.editor.SearchMode(m.canvas, m.status, m.tty, clearPreviousSearch, searchForward, undo)
			return m, nil
		}

		if m.editor != nil {
			undo.Snapshot(m.editor)
			m.editor.ClearSearch()

			// Handle markdown table editing
			if m.editor.mode == mode.Markdown && m.editor.InTable() && !m.keyHistory.PrevIs("c:23") {
				m.editor.GoToStartOfTextLine(m.canvas)
				const justFormat = true
				const displayQuickHelp = false
				m.editor.EditMarkdownTable(m.tty, m.canvas, m.status, m.bookmark, justFormat, displayQuickHelp)
				return m, nil
			} else if m.editor.mode == mode.Markdown && !m.keyHistory.PrevIs("c:23") {
				m.editor.GoToStartOfTextLine(m.canvas)
				m.editor.FormatAllMarkdownTables()
				return m, nil
			}

			// Format code
			m.status.ClearAll(m.canvas, true)
			m.editor.formatCode(m.canvas, m.tty, m.status, &m.jsonFormatToggle)

			if m.editor.AtOrAfterEndOfLine() {
				m.editor.End(m.canvas)
			}

			m.status.HoldMessage(m.canvas, 250*time.Millisecond)
		}

		m.redraw = true
		return m, nil

	case leftArrow, rightArrow, upArrow, downArrow:
		if m.editor != nil {
			m.handleArrowKeys(key)
		}
		m.redraw = true
		m.redrawCursor = true
		return m, nil

	case " ": // space
		if m.editor != nil && m.editor.readOnly && !m.editor.blockMode {
			// Scroll down if read-only
			redraw := m.editor.PgDn(m.canvas, m.status)
			m.redraw = redraw
			if !redraw {
				m.status.Clear(m.canvas, false)
				m.status.SetMessage(endOfFileMessage)
				m.status.Show(m.canvas, m.editor)
			}
			m.redrawCursor = true
			if m.editor.AfterLineScreenContents() {
				m.editor.End(m.canvas)
			}
			return m, nil
		}

		// Regular space insertion
		if m.editor != nil {
			undo.Snapshot(m.editor)
			wrapped := m.editor.InsertRune(m.canvas, ' ')
			if !wrapped {
				m.editor.WriteRune(m.canvas)
				m.editor.Next(m.canvas)
			}
			m.redraw = true
		}
		return m, nil

	case "c:13", "\n": // return
		if m.editor != nil {
			if m.editor.readOnly {
				m.status.Clear(m.canvas, false)
				m.status.SetMessage("Read only")
				m.status.Show(m.canvas, m.editor)
				return m, nil
			}

			undo.Snapshot(m.editor)
			m.editor.ReturnPressed(m.canvas, m.status)
		}
		return m, nil

	case pgUpKey: // page up
		if m.editor != nil {
			h := int(m.canvas.H())
			m.redraw = m.editor.ScrollUp(m.canvas, m.status, int(float64(h)*0.9))
			m.redrawCursor = true
			if m.editor.AfterLineScreenContents() {
				m.editor.End(m.canvas)
			}
			m.editor.drawProgress.Store(true)
			m.editor.drawFuncName.Store(true)
		}
		return m, nil

	case pgDnKey: // page down
		if m.editor != nil {
			h := int(m.canvas.H())
			redraw := m.editor.ScrollDown(m.canvas, m.status, int(float64(h)*0.9), h)
			m.redraw = redraw
			if !redraw {
				m.status.Clear(m.canvas, false)
				m.status.SetMessage(endOfFileMessage)
				m.status.Show(m.canvas, m.editor)
			}
			m.redrawCursor = true
			if m.editor.AfterLineScreenContents() {
				m.editor.End(m.canvas)
			}
			m.editor.drawProgress.Store(true)
			m.editor.drawFuncName.Store(true)
		}
		return m, nil

	case "c:8", "c:127": // ctrl-h or backspace
		if m.editor != nil {
			// Scroll up if read-only
			if m.editor.readOnly && !m.editor.blockMode {
				m.redraw = m.editor.ScrollUp(m.canvas, m.status, m.editor.pos.scrollSpeed*2)
				m.redrawCursor = true
				if m.editor.AfterLineScreenContents() {
					m.editor.End(m.canvas)
				}
				return m, nil
			}

			// Clear search if active
			if len(m.editor.SearchTerm()) > 0 {
				m.editor.ClearSearch()
				m.redraw = true
				m.redrawCursor = true
			}

			undo.Snapshot(m.editor)
			m.editor.Backspace(m.canvas, m.bookmark)
			m.redrawCursor = true
			m.redraw = true
		}
		return m, nil

	case "c:9": // tab
		if m.editor != nil {
			// Handle tab completion, spell check, debug mode, etc.
			m.handleTabKey()
			m.redrawCursor = true
			m.redraw = true
		}
		return m, nil

	case "c:27": // esc
		if m.editor != nil {
			m.editor.blockMode = false
			// Handle special cases like man page mode
			if m.editor.mode == mode.ManPage {
				return m, tea.Quit
			}
			// Exit debug mode if active
			if m.editor.debugMode {
				m.editor.DebugEnd()
				m.editor.debugMode = false
				m.status.SetMessageAfterRedraw("Normal mode")
				m.redraw = true
				m.redrawCursor = true
				return m, nil
			}
			// Reset state and redraw
			m.lastCopyY = -1
			m.lastPasteY = -1
			m.lastCutY = -1
			stopBackgroundProcesses()
			const drawLines = true
			m.editor.FullResetRedraw(m.canvas, m.status, drawLines, false)
			m.regularEditingRightNow = true
			if m.editor.macro != nil || m.editor.playBackMacroCount > 0 {
				m.editor.playBackMacroCount = 0
				m.editor.macro = nil
				m.status.SetMessageAfterRedraw("Macro cleared")
			}
			m.redraw = true
			m.redrawCursor = true
		}
		return m, nil

	default:
		// Handle other keys (letters, symbols, etc.)
		if m.editor != nil {
			m.handleRegularKey(key)
		}
	}

	// Update key history
	if m.clearKeyHistory {
		m.keyHistory.Clear()
		m.clearKeyHistory = false
	} else {
		m.keyHistory.Push(key)
	}

	m.redraw = true
	m.redrawCursor = true
	return m, nil
}

// convertKeyMsg converts bubbletea KeyMsg to Orbiton's key format
func (m OrbitonModel) convertKeyMsg(msg tea.KeyMsg) string {
	switch msg.Type {
	case tea.KeyCtrlC:
		return "c:3"
	case tea.KeyCtrlQ:
		return "c:17"
	case tea.KeyCtrlW:
		return "c:23"
	case tea.KeyCtrlS:
		return "c:19"
	case tea.KeyCtrlF:
		return "c:6"
	case tea.KeyCtrlG:
		return "c:7"
	case tea.KeyCtrlN:
		return "c:14"
	case tea.KeyCtrlP:
		return "c:16"
	case tea.KeyCtrlL:
		return "c:12"
	case tea.KeyCtrlO:
		return "c:15"
	case tea.KeyCtrlT:
		return "c:20"
	case tea.KeyCtrlK:
		return "c:11"
	case tea.KeyCtrlX:
		return "c:24"
	case tea.KeyCtrlV:
		return "c:22"
	case tea.KeyCtrlU:
		return "c:21"
	case tea.KeyCtrlZ:
		return "c:26"
	case tea.KeyCtrlR:
		return "c:18"
	case tea.KeyCtrlB:
		return "c:2"
	case tea.KeyCtrlJ:
		return "c:10"
	case tea.KeyCtrlA:
		return "c:1"
	case tea.KeyCtrlE:
		return "c:5"
	case tea.KeyCtrlD:
		return "c:4"
	case tea.KeyCtrlH:
		return "c:8"
	case tea.KeyBackspace:
		return "c:127"
	case tea.KeySpace:
		return " "
	case tea.KeyEnter:
		return "c:13"
	case tea.KeyTab:
		return "c:9"
	case tea.KeyEsc:
		return "c:27"
	case tea.KeyUp:
		return upArrow
	case tea.KeyDown:
		return downArrow
	case tea.KeyLeft:
		return leftArrow
	case tea.KeyRight:
		return rightArrow
	case tea.KeyPgUp:
		return pgUpKey
	case tea.KeyPgDown:
		return pgDnKey
	case tea.KeyHome:
		return homeKey
	case tea.KeyEnd:
		return endKey
	case tea.KeyRunes:
		return string(msg.Runes)
	default:
		// Handle regular character input
		if len(msg.Runes) > 0 {
			return string(msg.Runes)
		}
		return ""
	}
}

// handleArrowKeys processes arrow key input with held-down acceleration
func (m OrbitonModel) handleArrowKeys(key string) {
	if m.editor == nil {
		return
	}

	switch key {
	case leftArrow:
		// Check if it's a special case for command prompt
		if m.keyHistory.SpecialArrowKeypressWith(leftArrow) {
			m.editor.Up(m.canvas, m.status)
			m.editor.Up(m.canvas, m.status)
			m.editor.CommandPrompt(m.canvas, m.tty, m.status, m.bookmark, undo)
			m.keyHistory.Clear()
			return
		}

		m.editor.CursorBackward(m.canvas, m.status)

		// Handle held down acceleration
		if m.keyHistory.TwoLastAre(leftArrow) && m.keyHistory.AllWithin(200*time.Millisecond) && m.keyHistory.LastChanged(200*time.Millisecond) {
			if m.heldDownLeftArrowTime.IsZero() {
				m.heldDownLeftArrowTime = time.Now()
			}
			heldDuration := time.Since(m.heldDownLeftArrowTime)
			steps := int(int64(heldDuration) / int64(delayUntilSpeedUp))
			for i := 1; i < steps; i++ {
				m.editor.CursorBackward(m.canvas, m.status)
			}
		}

	case rightArrow:
		if m.keyHistory.SpecialArrowKeypressWith(rightArrow) {
			m.editor.Up(m.canvas, m.status)
			m.editor.Up(m.canvas, m.status)
			m.editor.CommandPrompt(m.canvas, m.tty, m.status, m.bookmark, undo)
			m.keyHistory.Clear()
			return
		}

		m.editor.CursorForward(m.canvas, m.status)

		if m.keyHistory.TwoLastAre(rightArrow) && m.keyHistory.AllWithin(200*time.Millisecond) && m.keyHistory.LastChanged(200*time.Millisecond) {
			if m.heldDownRightArrowTime.IsZero() {
				m.heldDownRightArrowTime = time.Now()
			}
			heldDuration := time.Since(m.heldDownRightArrowTime)
			steps := int(int64(heldDuration) / int64(delayUntilSpeedUp))
			for i := 1; i < steps; i++ {
				m.editor.CursorForward(m.canvas, m.status)
			}
		}

	case upArrow:
		if m.keyHistory.SpecialArrowKeypressWith(upArrow) {
			m.editor.CommandPrompt(m.canvas, m.tty, m.status, m.bookmark, undo)
			m.keyHistory.Clear()
			return
		}

		m.editor.CursorUpward(m.canvas, m.status)

		if m.keyHistory.TwoLastAre(upArrow) && m.keyHistory.AllWithin(200*time.Millisecond) && m.keyHistory.LastChanged(200*time.Millisecond) {
			m.editor.CursorUpward(m.canvas, m.status)
		}

	case downArrow:
		if m.keyHistory.SpecialArrowKeypressWith(downArrow) {
			m.editor.CommandPrompt(m.canvas, m.tty, m.status, m.bookmark, undo)
			m.keyHistory.Clear()
			return
		}

		m.editor.CursorDownward(m.canvas, m.status)

		if m.keyHistory.TwoLastAre(downArrow) && m.keyHistory.AllWithin(200*time.Millisecond) && m.keyHistory.LastChanged(200*time.Millisecond) {
			m.editor.CursorDownward(m.canvas, m.status)
		}

		if m.editor.AfterLineScreenContents() || m.editor.AfterEndOfLine() {
			m.editor.End(m.canvas)
		}
	}

	// Note: highlighting state updates are handled by the main update loop
}

// handleRegularKey processes regular character input
func (m OrbitonModel) handleRegularKey(key string) {
	if m.editor == nil {
		return
	}

	keyRunes := []rune(key)
	if len(keyRunes) == 0 {
		return
	}

	r := keyRunes[0]

	// Handle special quit sequences
	if r == 'q' && m.editor.mode == mode.ManPage {
		// Quit will be handled by the main update loop
		return
	}

	// Handle escape sequences like esc,q and esc,w
	if r == 'q' && !m.editor.nanoMode.Load() && m.keyHistory.PrevPrev() == "c:27" && m.keyHistory.Prev() == "," {
		m.editor.Backspace(m.canvas, m.bookmark)
		// Quit will be handled by the main update loop
		return
	} else if r == 'w' && !m.editor.nanoMode.Load() && m.keyHistory.PrevPrev() == "c:27" && m.keyHistory.Prev() == "," {
		m.editor.Backspace(m.canvas, m.bookmark)
		m.editor.UserSave(m.canvas, m.tty, m.status)
		return
	}

	// Regular character input
	undo.Snapshot(m.editor)

	// Handle smart dedenting for closing brackets
	if r == '}' || r == ']' || r == ')' {
		m.handleSmartDedent(r)
	}

	// Insert the character
	wrapped := m.editor.InsertRune(m.canvas, r)
	m.editor.WriteRune(m.canvas)
	if !wrapped {
		m.editor.Next(m.canvas)
	}
}

// handleSmartDedent handles automatic dedenting for closing brackets
func (m OrbitonModel) handleSmartDedent(r rune) {
	if m.editor == nil {
		return
	}

	noContentHereAlready := len(m.editor.TrimmedLine()) == 0
	leadingWhitespace := m.editor.LeadingWhitespace()
	nextLineContents := m.editor.Line(m.editor.DataY() + 1)
	currentX := m.editor.pos.sx

	var foundBracketBelow bool
	switch r {
	case '}':
		foundBracketBelow = currentX-1 == strings.Index(nextLineContents, "}")
	case ']':
		foundBracketBelow = currentX-1 == strings.Index(nextLineContents, "]")
	case ')':
		foundBracketBelow = currentX-1 == strings.Index(nextLineContents, ")")
	}

	// Dedent if conditions are met
	if !foundBracketBelow && m.editor.pos.sx > 0 && len(leadingWhitespace) > 0 && noContentHereAlready {
		newLeadingWhitespace := leadingWhitespace
		if strings.HasSuffix(leadingWhitespace, "\t") {
			newLeadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-1]
			m.editor.pos.sx -= m.editor.indentation.PerTab
		} else if strings.HasSuffix(leadingWhitespace, strings.Repeat(" ", m.editor.indentation.PerTab)) {
			newLeadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-m.editor.indentation.PerTab]
			m.editor.pos.sx -= m.editor.indentation.PerTab
		}
		m.editor.SetCurrentLine(newLeadingWhitespace)
	}
}

// handleTabKey processes tab key functionality including smart indentation and code completion
func (m OrbitonModel) handleTabKey() {
	if m.editor == nil {
		return
	}

	// Handle spell check mode
	if m.editor.spellCheckMode {
		if ignoredWord := m.editor.RemoveCurrentWordFromWordList(); ignoredWord != "" {
			typo, corrected := m.editor.NanoNextTypo(m.canvas, m.status)
			msg := "Ignored " + ignoredWord
			if spellChecker != nil && typo != "" {
				msg += ". Found " + typo
				if corrected != "" {
					msg += " which could be " + corrected + "."
				} else {
					msg += "."
				}
			}
			m.status.SetMessageAfterRedraw(msg)
		}
		return
	}

	// Handle debug mode
	if m.editor.debugMode {
		m.editor.debugStepInto = !m.editor.debugStepInto
		return
	}

	y := int(m.editor.DataY())
	r := m.editor.Rune()
	leftRune := m.editor.LeftRune()

	// Tab triggered code completion with Ollama
	if cc.Loaded() && m.editor.mode != mode.Blank && m.editor.AnyTextBeforeCursor() {
		m.status.ClearAll(m.canvas, true)
		m.status.SetMessage(fmt.Sprintf("Generating code with Ollama and the %s model...", cc.ModelName))
		m.status.ShowNoTimeout(m.canvas, m.editor)
		ShowCursor(false)

		linesOfContext := codeCompletionContextLines / 2
		currentLineIndex := int(m.editor.LineIndex())

		var codeBefore strings.Builder
		for i := currentLineIndex - linesOfContext; i < currentLineIndex; i++ {
			if i < 0 {
				continue
			}
			codeBefore.WriteString(m.editor.Line(LineIndex(i)) + "\n")
		}
		codeBefore.WriteString(m.editor.CurrentLine())

		var codeAfter strings.Builder
		for i := currentLineIndex + 1; i < currentLineIndex+linesOfContext; i++ {
			if int(i) >= m.editor.Len() {
				break
			}
			codeAfter.WriteString(m.editor.Line(LineIndex(i)) + "\n")
		}

		codeStart := codeBefore.String()
		codeEnd := codeAfter.String()

		if responseString, err := cc.CompleteBetween(codeStart, codeEnd); err == nil {
			generatedCodeCompletion := strings.TrimSuffix(responseString, "\n")
			if generatedCodeCompletion != "" {
				undo.Snapshot(m.editor)
				m.editor.InsertStringAndMove(m.canvas, generatedCodeCompletion)
			}
			m.redraw = true
			m.redrawCursor = true
		}
		return
	}

	// Smart indentation logic
	trimmedLine := m.editor.TrimmedLine()
	endsWithSpecial := len(trimmedLine) > 1 && (r == '{' || r == '(' || r == '[' || r == ':')
	noSmartIndentation := m.editor.NoSmartIndentation()

	if (!unicode.IsSpace(leftRune) || endsWithSpecial) && m.editor.pos.sx > 0 && !noSmartIndentation {
		lineAbove := 1
		if strings.TrimSpace(m.editor.Line(LineIndex(y-lineAbove))) == "" {
			lineAbove--
		}
		indexAbove := LineIndex(y - lineAbove)

		if strings.TrimSpace(m.editor.Line(indexAbove)) != "" {
			undo.Snapshot(m.editor)

			var (
				spaceAbove        = m.editor.LeadingWhitespaceAt(indexAbove)
				strippedLineAbove = m.editor.StripSingleLineComment(strings.TrimSpace(m.editor.Line(indexAbove)))
				newLeadingSpace   string
			)

			oneIndentation := m.editor.indentation.String()

			// Smart indentation logic
			if !strings.HasPrefix(strippedLineAbove, "switch ") && strings.HasPrefix(strippedLineAbove, "case ") ||
				strings.HasSuffix(strippedLineAbove, "{") || strings.HasSuffix(strippedLineAbove, "[") ||
				strings.HasSuffix(strippedLineAbove, "(") || strings.HasSuffix(strippedLineAbove, ":") ||
				strings.HasSuffix(strippedLineAbove, " \\") || strings.HasPrefix(strippedLineAbove, "if ") {
				newLeadingSpace = spaceAbove + oneIndentation
			} else if ((len(spaceAbove) - len(oneIndentation)) > 0) && strings.HasSuffix(trimmedLine, "}") {
				newLeadingSpace = spaceAbove[:len(spaceAbove)-len(oneIndentation)]
			} else {
				newLeadingSpace = spaceAbove
			}

			m.editor.SetCurrentLine(newLeadingSpace + trimmedLine)
			if m.editor.AtOrAfterEndOfLine() {
				m.editor.End(m.canvas)
			}
			return
		}
	}

	// Regular tab insertion
	undo.Snapshot(m.editor)
	if m.editor.indentation.Spaces {
		for i := 0; i < m.editor.indentation.PerTab; i++ {
			m.editor.InsertRune(m.canvas, ' ')
			m.editor.WriteTab(m.canvas)
			m.editor.Next(m.canvas)
		}
	} else {
		m.editor.InsertRune(m.canvas, '\t')
		m.editor.WriteTab(m.canvas)
		m.editor.Next(m.canvas)
	}
}

// NewOrbitonModel creates a new bubbletea model for Orbiton
func NewOrbitonModel(tty *TTY, fnord FilenameOrData, lineNumber LineNumber, colNumber ColNumber, forceFlag bool, theme Theme, syntaxHighlight, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp, fmtFlag bool) (*OrbitonModel, error) {
	// Create canvas
	canvas := NewCanvas()
	canvas.ShowCursor()

	// Create editor
	editor, messageAfterRedraw, displayedImage, err := NewEditor(tty, canvas, fnord, lineNumber, colNumber, theme, syntaxHighlight, true, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp)
	if err != nil {
		return nil, fmt.Errorf("failed to create editor: %v", err)
	}

	if displayedImage {
		// Special case for image display
		return nil, fmt.Errorf("image display not supported in bubbletea mode")
	}

	// Create status bar
	statusDuration := 2700 * time.Millisecond
	status := editor.NewStatusBar(statusDuration, messageAfterRedraw)

	// Initialize model
	model := &OrbitonModel{
		editor:                     editor,
		canvas:                     canvas,
		status:                     status,
		tty:                        tty,
		fnord:                      fnord,
		lineNumber:                 lineNumber,
		colNumber:                  colNumber,
		forceFlag:                  forceFlag,
		theme:                      theme,
		syntaxHighlight:            syntaxHighlight,
		monitorAndReadOnly:         monitorAndReadOnly,
		nanoMode:                   nanoMode,
		createDirectoriesIfMissing: createDirectoriesIfMissing,
		displayQuickHelp:           displayQuickHelp,
		noDisplayQuickHelp:         noDisplayQuickHelp,
		fmtFlag:                    fmtFlag,
		keyHistory:                 NewKeyHistory(),
		lastCopyY:                  -1,
		lastPasteY:                 -1,
		lastCutY:                   -1,
		firstPasteAction:           true,
		firstCopyAction:            true,
		regularEditingRightNow:     true,
	}

	return model, nil
}
