package main

import (
	"fmt"
	"strings"
)

// TutorialStep represents a step in the tutorial wizard
type TutorialStep struct {
	title       string
	description string
	expectKeys  []string
}

// Tutorial is a collection of steps
type Tutorial []TutorialStep

var tutorialSteps = Tutorial{
	TutorialStep{
		title:       "Start of text",
		description: "Press ctrl-a to go to the start of the text on the current line.",
		expectKeys:  []string{"c:1"}, // ctrl-a
	},
	TutorialStep{
		title:       "Start of line",
		description: "Press ctrl-a twice to go to the start of the line.",
		expectKeys:  []string{"c:1", "c:1"}, // ctrl-a
	},
	TutorialStep{
		title:       "End of the line above",
		description: "Press ctrl-a three times to go to the end of the line above.",
		expectKeys:  []string{"c:1", "c:1", "c:1"}, // ctrl-a
	},
	TutorialStep{
		title:       "Create a bookmark",
		description: "Press ctrl-b to bookmark the current line.",
		expectKeys:  []string{"c:2"}, // ctrl-b
	},
	TutorialStep{
		title:       "Jump to a bookmark",
		description: "Press ctrl-b to jump to the bookmark. It must be on a different line than the current line.",
		expectKeys:  []string{"c:2"}, // ctrl-b
	},
	TutorialStep{
		title:       "Remove a bookmark",
		description: "Press ctrl-b to clear a bookmark. The bookmark must be set and the current line must be the bookmarked line.",
		expectKeys:  []string{"c:2"}, // ctrl-b
	},
	TutorialStep{
		title:       "Copy line",
		description: "Press ctrl-c to copy the current line to the clipboard. If no clipboard is available, an internal buffer is used.",
		expectKeys:  []string{"c:3"}, // ctrl-c
	},
	TutorialStep{
		title:       "Copy block of text",
		description: "Press ctrl-c twice to copy a block of text (until the next blank line) to the clipboard. If no clipboard is available, an internal buffer is used. For some terminal emulators, this must not be pressed too fast.",
		expectKeys:  []string{"c:3", "c:3"}, // ctrl-c
	},
	TutorialStep{
		title:       "Delete the current letter",
		description: "Press ctrl-d to delete the current letter.",
		expectKeys:  []string{"c:8"}, // ctrl-d
	},
	TutorialStep{
		title:       "End of the line",
		description: "Press ctrl-e to go to the end of the line.",
		expectKeys:  []string{"c:5"}, // ctrl-e
	},
	TutorialStep{
		title:       "Start of the next line",
		description: "Press ctrl-e twice to go to the start of the next line.",
		expectKeys:  []string{"c:5"}, // ctrl-e
	},
	TutorialStep{
		title:       "Search",
		description: "Press ctrl-f and type in the text to search for.",
		expectKeys:  []string{"c:6"}, // ctrl-f
	},
	TutorialStep{
		title:       "Search and replace",
		description: "Press ctrl-f, type in text to search for, press Tab, type in text that all instances should be replaced with and then press return.",
		expectKeys:  []string{"c:6"}, // ctrl-f
	},
	TutorialStep{
		title:       "Go to definition",
		description: "For some programming languages, ctrl-g can be pressed to jump to a definition, and ctrl-b can be used to jump back.",
		expectKeys:  []string{"c:7"}, // ctrl-g
	},
	TutorialStep{
		title:       "Toggle block editing mode",
		description: "ctrl-g will toggle block editing mode, where multiple lines in a block (until a blank line of EOF) can be edited at once. An informative status bar will also be shown.",
		expectKeys:  []string{"c:7"}, // ctrl-g
	},
	TutorialStep{
		title:       "Delete to letter to the left",
		description: "Press ctrl-h or backspace to delete the letter to the left.",
		expectKeys:  []string{"c:8"}, // ctrl-h or backspace
	},
	TutorialStep{
		title:       "Indentation",
		description: "Press ctrl-i or tab to indent a line.",
		expectKeys:  []string{"c:9"}, // ctrl-i or tab
	},
	TutorialStep{
		title:       "Join",
		description: "Press ctrl-j to join this line with the next. The next line is placed after the current one.",
		expectKeys:  []string{"c:10"}, // ctrl-j
	},
	TutorialStep{
		title:       "Remove the rest of the line",
		description: "Press ctrl-k to remove the rest of the current line.",
		expectKeys:  []string{"c:11"}, // ctrl-k
	},
	TutorialStep{
		title:       "Jump to a location",
		description: "Press ctrl-l and then press one of the highlighted letters to jump there.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the top",
		description: "Unless the cursor is already at the top, press ctrl-l and then return.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the top (method 2)",
		description: "Press ctrl-l and then t to jump to the top of the file.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the top (method 3)",
		description: "Press ctrl-l, type in 0 and press return.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the bottom",
		description: "Press ctrl-l and then b to jump to the bottom of the file.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the bottom (method 2)",
		description: "Press ctrl-l and return to jump to the top, then ctrl-l and return to jump to the bottom.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the bottom (method 3)",
		description: "Press ctrl-l, type in 100% and press return.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the bottom (method 4)",
		description: "Press ctrl-l, type in 1. and press return.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the center of the file",
		description: "Press ctrl-l and then c to jump to the center of the file.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the center of the file (method 2)",
		description: "Press ctrl-l, type in 50% and press return.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Jump to the center of the file (method 3)",
		description: "Press ctrl-l, type in .5 and press return.",
		expectKeys:  []string{"c:12"}, // ctrl-l
	},
	TutorialStep{
		title:       "Move a line down or start a new line",
		description: "Press ctrl-m or press return.",
		expectKeys:  []string{"c:13"}, // ctrl-m
	},
	TutorialStep{
		title:       "Move down 10 lines",
		description: "Press ctrl-n to move and scroll down 10 lines.",
		expectKeys:  []string{"c:14"}, // ctrl-n
	},
	TutorialStep{
		title:       "Go to the next instance",
		description: "When searching, press ctrl-n to go to the next instance.",
		expectKeys:  []string{"c:14"}, // ctrl-n
	},
	TutorialStep{
		title:       "Go to the next instruction",
		description: "When debugging, press ctrl-n to go to the next instruction.",
		expectKeys:  []string{"c:14"}, // ctrl-n
	},
	TutorialStep{
		title:       "Main menu",
		description: "Press ctrl-o to open the main menu.",
		expectKeys:  []string{"c:15"}, // ctrl-o
	},
	TutorialStep{
		title:       "Move up 10 lines",
		description: "Press ctrl-p to move and scroll up 10 lines.",
		expectKeys:  []string{"c:16"}, // ctrl-p
	},
	TutorialStep{
		title:       "Go to the previous instance",
		description: "When searching, press ctrl-p to go to the previous instance.",
		expectKeys:  []string{"c:16"}, // ctrl-p
	},
	TutorialStep{
		title:       "Cycle register pane layout",
		description: "When debugging, press ctrl-p to cycle the size of the register pane: small -> large -> hidden",
		expectKeys:  []string{"c:16"}, // ctrl-p
	},
	TutorialStep{
		title:       "Quit",
		description: "Press ctrl-q to quit, no questions asked.",
		expectKeys:  []string{"c:17"}, // ctrl-q
	},
	TutorialStep{
		title:       "Open a portal",
		description: "Press ctrl-r to open a portal that can be used to paste lines into another file with ctrl-v.",
		expectKeys:  []string{"c:18"}, // ctrl-r
	},
	TutorialStep{
		title:       "Close a portal",
		description: "If a portal is open, it will time out after 20 minutes, or it can be closed with ctrl-r.",
		expectKeys:  []string{"c:18"}, // ctrl-r
	},
	TutorialStep{
		title:       "Save",
		description: "Press ctrl-s to save the current file.",
		expectKeys:  []string{"c:19"}, // ctrl-s
	},
	TutorialStep{
		title:       "Save without using the pinky finger",
		description: "In rapid succession, press arrow right, arrow down and arrow left. When \"o:\" appears at the bottom, press arrow down to save.",
		expectKeys:  []string{}, // tbd
	},
	TutorialStep{
		title:       "Save and quit without using the pinky finger",
		description: "In rapid succession, press arrow right, arrow down and arrow left. When \"o:\" appears at the bottom, press arrow up to save and quit.",
		expectKeys:  []string{}, // tbd
	},
	TutorialStep{
		title:       "Record macro",
		description: "Press ctrl-t to record keypresses. Press ctrl-t again to stop recording.",
		expectKeys:  []string{"c:20"}, // ctrl-t
	},
	TutorialStep{
		title:       "Play back macro",
		description: "Press ctrl-t to play back the current macro.",
		expectKeys:  []string{"c:20"}, // ctrl-t
	},
	TutorialStep{
		title:       "Clear macro",
		description: "Press Esc to clear the current macro.",
		expectKeys:  []string{"c:27"}, // esc
	},
	TutorialStep{
		title:       "Toggle checkbox",
		description: "When editing Markdown, move the cursor to a line with a checkbox (\"- [ ] TODO\") and press ctrl-t or ctrl-space to toggle it.",
		expectKeys:  []string{"c:20"}, // ctrl-t
	},
	TutorialStep{
		title:       "Edit table",
		description: "When editing Markdown, move the cursor to a line with a table and press ctrl-t to enter the table editor. This feature currently only supports small tables.",
		expectKeys:  []string{"c:20"}, // ctrl-t
	},
	TutorialStep{
		title:       "Format table",
		description: "When editing Markdown, move the cursor to a line with a table and press ctrl-w to format it.",
		expectKeys:  []string{"c:23"}, // ctrl-w
	},
	TutorialStep{
		title:       "Undo",
		description: "Press ctrl-u to undo the last operation. There is no redo, yet.",
		expectKeys:  []string{"c:21"}, // ctrl-u
	},
	TutorialStep{
		title:       "Paste",
		description: "Press ctrl-v to paste the first line from the clipboard. The string will be trimmed.",
		expectKeys:  []string{"c:22"}, // ctrl-v
	},
	TutorialStep{
		title:       "Paste",
		description: "Press ctrl-v twice to paste the contents of the clipbard.",
		expectKeys:  []string{"c:22", "c:22"}, // ctrl-v
	},
	TutorialStep{
		title:       "Generate template",
		description: "Open an empty source code file and press ctrl-w to generate a \"hello world\" program. This applies to several programming languages.",
		expectKeys:  []string{"c:23"}, // ctrl-w
	},
	TutorialStep{
		title:       "Format source code",
		description: "Open a source code file and press ctrl-w to format it. This applies to several programming languages.",
		expectKeys:  []string{"c:23"}, // ctrl-w
	},
	TutorialStep{
		title:       "Cut",
		description: "Press ctrl-x to cut the current line and place it in the clipboard.",
		expectKeys:  []string{"c:24"}, // ctrl-x
	},
	TutorialStep{
		title:       "Paste",
		description: "Press ctrl-x twice to cut the current block of text (to a blank line) and place it in the clipboard.",
		expectKeys:  []string{"c:24", "c:24"}, // ctrl-x
	},
	TutorialStep{
		title:       "Go to start of line (method 2)",
		description: "Press ctrl-y to go to the start of the text, then line, then the end of the line above. Same as ctrl-a.",
		expectKeys:  []string{"c:25"}, // ctrl-y
	},
	TutorialStep{
		title:       "Undo (method 2)",
		description: "Press ctrl-z to undo the last operation. If ctrl-z backgrounds the application, run \"fg\" to bring it back.",
		expectKeys:  []string{"c:26"}, // ctrl-z
	},
	TutorialStep{
		title:       "Build source code",
		description: "Open a source code file and press ctrl-space to build it. This works for some projects and programming languages.",
		expectKeys:  []string{"c:0"}, // ctrl-space
	},
	TutorialStep{
		title:       "Build and run",
		description: "Open a source code file and press ctrl-space twice to build it, run it and also display stdout + stderr.",
		expectKeys:  []string{"c:0", "c:0"}, // ctrl-space
	},
	TutorialStep{
		title:       "Insert a file",
		description: "Let the file be named include.txt, then select the 'Insert \"include.txt\" at the current line' option in the ctrl-o menu.",
		expectKeys:  []string{}, // tbd
	},
	TutorialStep{
		title:       "Insert a file (method 2)",
		description: "In rapid succession, press arrow right, arrow down and arrow left. Then type \"insertfile somefile.txt\" to insert somefile.txt into the current file.",
		expectKeys:  []string{}, // tbd
	},
	TutorialStep{
		title:       "English spell check (experimental feature)",
		description: "Press ctrl-f, type t and press return. Then press ctrl-n for next instance, ctrl-a to add the word temporarily or ctrl-i to ignore the word temporarily.",
		expectKeys:  []string{}, // tbd
	},
	TutorialStep{
		title:       "Jump to matching parenthesis",
		description: "Be on a (, [, {, }, ] or ) character. Press ctrl-_ to jump to the matching one, for instance the next \")\" if the cursor is on \"(\".",
		expectKeys:  []string{"c:31"}, // ctrl-_
	},
	TutorialStep{
		title:       "Insert digraph",
		description: "Press ctrl-_ to insert a digraph. For instance \"ae\" to insert \"æ\". These are the same as for ViM or NeoViM. Do not be on a (, [, {, }, ] or ) character.",
		expectKeys:  []string{"c:31"}, // ctrl-_
	},
	TutorialStep{
		title:       "Tutorial complete",
		description: "Press q, esc or ctrl-q to end this tutorial.",
		expectKeys:  []string{"c:32"}, // space
	},
}

// LaunchTutorial launches a short and sweet tutorial that covers at least portals and cut/paste
func LaunchTutorial(tty *TTY, c *Canvas, e *Editor, status *StatusBar) {
	const repositionCursorAfterDrawing = false
	const marginX = 4

	minWidth := 32
	for _, step := range tutorialSteps {
		for _, line := range strings.Split(step.description, "\n") {
			if len(line) > minWidth {
				minWidth = len(line) + marginX
			}
		}
	}

	displayedStatusOnce := false

	i := 0
	for {
		if i == 0 && !displayedStatusOnce {
			status.SetMessage("q to end")
			status.Show(c, e)
			displayedStatusOnce = true
		} else {
			status.Clear(c, false)
		}

		step := tutorialSteps[i]
		progress := fmt.Sprintf("%d / %d", i+1, len(tutorialSteps))
		step.Draw(c, e, progress, minWidth, repositionCursorAfterDrawing)

		// Wait for a keypress
		key := tty.String()
		switch key {
		case " ", "c:13", "↓", "→", "j", "c:14", "n": // space, return, down, right, j, ctrl-n or n to go to the next step
			if i < (len(tutorialSteps) - 1) {
				i++
			}
			continue
		case "↑", "←", "k", "c:16", "p": // up, left, k, ctrl-p or p to go to the previous step
			if i > 0 {
				i--
			}
			continue
		case "c:17", "c:27", "q", "x": // ctrl-q, esc, q or x to exit
			return
		}
		// Other keypress, do nothing
	}
}

// Draw draws a step of the tutorial
func (step TutorialStep) Draw(c *Canvas, e *Editor, progress string, minWidth int, repositionCursorAfterDrawing bool) {
	canvasBox := NewCanvasBox(c)

	// Window is the background box that will be drawn in the upper right
	centerBox := NewBox()

	centerBox.EvenLowerRightPlacement(canvasBox, minWidth)
	e.redraw.Store(true)

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(centerBox, 2, 2)

	// Get the current theme for the register box
	bt := e.NewBoxTheme()
	bt.Foreground = &e.BoxTextColor
	bt.Background = &e.DebugInstructionsBackground

	// First figure out how many lines of text this will be after word wrap
	const dryRun = true
	addedLines := e.DrawText(bt, c, listBox, step.description, dryRun)

	if addedLines > listBox.H {
		// Then adjust the box height and text position (addedLines could very well be 0)
		centerBox.Y -= addedLines
		centerBox.H += addedLines
		listBox.Y -= addedLines
	}

	// Then draw the box with the text
	e.DrawBox(bt, c, centerBox)
	e.DrawTitle(bt, c, centerBox, step.title, true)
	e.DrawFooter(bt, c, centerBox, "("+progress+")")
	e.DrawText(bt, c, listBox, step.description, false)

	// Blit
	c.HideCursorAndDraw()

	// Reposition the cursor, if needed
	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}
