package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/xyproto/vt100"
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
		description: "Press ctrl-a or ctrl-y to go to the start of the text on the current line.",
		expectKeys:  []string{"c:1"}, // ctrl-a
	},
	TutorialStep{
		title:       "Start of line",
		description: "Press ctrl-a or ctrl-y twice to go to the start of the line.",
		expectKeys:  []string{"c:1", "c:1"}, // ctrl-a
	},
	TutorialStep{
		title:       "End of the line above",
		description: "Press ctrl-a 3 times or ctrl-y 3 times to got to the end of the line above.",
		expectKeys:  []string{"c:1", "c:1", "c:1"}, // ctrl-a
	},
	TutorialStep{
		title:       "End of the line",
		description: "Press ctrl-e to go to the end of the line.",
		expectKeys:  []string{"c:5"}, // ctrl-e
	},
	TutorialStep{
		title:       "Delete the current letter",
		description: "Press ctrl-d to delete the current letter.",
		expectKeys:  []string{"c:8"}, // ctrl-g
	},
	TutorialStep{
		title:       "Delete to letter to the left",
		description: "Press ctrl-h or backspace to delete the letter to the left.",
		expectKeys:  []string{"c:9"}, // ctrl-h or backspace
	},
	TutorialStep{
		title:       "Insert template",
		description: "Open an empty source code file, like main.c, then press ctrl-w to insert a \"hello world\" program.",
		expectKeys:  []string{"c:23"}, // ctrl-w
	},
	TutorialStep{
		title:       "Format source code",
		description: "Edit a source code file, like main.c, then press ctrl-w to format the source code in an opinionated way.",
		expectKeys:  []string{"c:23"}, // ctrl-w
	},
	TutorialStep{
		title:       "Build source code",
		description: "Open a source code file, then press ctrl-space to try to build it. This only works for some projects.",
		expectKeys:  []string{"c:0"}, // ctrl-space
	},
	TutorialStep{
		title:       "Build and run",
		description: "Open a source code file, then build it, run it and display stdout by pressing ctrl-space twice.",
		expectKeys:  []string{"c:0", "c:0"}, // ctrl-space
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
		title:       "Record macro",
		description: "Press ctrl-t to start recording keypresses.",
		expectKeys:  []string{"c:20"}, // ctrl-t
	},
	TutorialStep{
		title:       "Stop recording macro",
		description: "Press ctrl-t to stop recording a macro.",
		expectKeys:  []string{"c:20"}, // ctrl-t
	},
	TutorialStep{
		title:       "Play back macro",
		description: "Press ctrl-t to play back a recorded macro.",
		expectKeys:  []string{"c:20"}, // ctrl-t
	},
	TutorialStep{
		title:       "Tab",
		description: "Press Tab or ctrl-i to indent a line or insert a tab character.",
		expectKeys:  []string{"c:9"}, // tab or ctrl-i
	},
	TutorialStep{
		title:       "Undo",
		description: "Press ctrl-u or ctrl-z to undo the last operation. If ctrl-z backgrounds the application, run \"fg\" to bring it back.",
		expectKeys:  []string{"c:21"}, // ctrl-u
	},
	TutorialStep{
		title:       "Go up 10 lines",
		description: "Press ctrl-p to move and scroll up 10 lines.",
		expectKeys:  []string{"c:16"}, // ctrl-p
	},
	TutorialStep{
		title:       "Go down 10 lines",
		description: "Press ctrl-n to move and scroll down 10 lines.",
		expectKeys:  []string{"c:14"}, // ctrl-n
	},
	TutorialStep{
		title:       "Find",
		description: "Press ctrl-f to search for text.",
		expectKeys:  []string{"c:6"}, // ctrl-f
	},
	TutorialStep{
		title:       "Tutorial complete",
		description: "",
		expectKeys:  []string{"c:32"}, // space
	},
}

// LaunchTutorial launches a short and sweet tutorial that covers at least portals and cut/paste
func LaunchTutorial(c *vt100.Canvas, e *Editor) {
	const repositionCursorAfterDrawing = false
	for i, step := range tutorialSteps {
		progress := fmt.Sprintf("%d / %d", i+1, len(tutorialSteps))
		step.Draw(c, e, progress, repositionCursorAfterDrawing)
		time.Sleep(1 * time.Second)
	}
}

// Draw draws a step of the tutorial
func (step TutorialStep) Draw(c *vt100.Canvas, e *Editor, progress string, repositionCursorAfterDrawing bool) {
	canvasBox := NewCanvasBox(c)

	// Window is the background box that will be drawn in the upper right
	centerBox := NewBox()

	minWidth := 32

	centerBox.EvenLowerRightPlacement(canvasBox, minWidth)
	e.redraw = true

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(centerBox, 2, 2)

	// Get the current theme for the register box
	bt := e.NewBoxTheme()
	bt.Foreground = &e.ListTextColor
	bt.Background = &e.DebugInstructionsBackground

	lines := strings.Split(step.description, "\n")

	e.DrawBox(bt, c, centerBox)
	e.DrawTitle(bt, c, centerBox, step.title)
	e.DrawFooter(bt, c, centerBox, "("+progress+")")
	e.DrawText(bt, c, listBox, lines)

	// Blit
	c.Draw()

	// Reposition the cursor, if needed
	if repositionCursorAfterDrawing {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}
}
