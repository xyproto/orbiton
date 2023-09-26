package main

import (
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

type Tutorial []TutorialStep

var tutorialSteps = Tutorial{
	TutorialStep{
		title:       "Go to start of text",
		description: "Press ctrl-a to go to the start of the text on the current line.",
		expectKeys:  []string{"c:1"}, // ctrl-a
	},
	TutorialStep{
		title:       "Go to start of line",
		description: "Press ctrl-a twice to go to the start of the line.",
		expectKeys:  []string{"c:1", "c:1"}, // ctrl-a
	},
	TutorialStep{
		title:       "Go to the end of the line above",
		description: "Press ctrl-a 3 times.",
		expectKeys:  []string{"c:1", "c:1", "c:1"}, // ctrl-a
	},
	TutorialStep{
		title:       "Go to the end of the line",
		description: "Press ctrl-e to go to the end of the line",
		expectKeys:  []string{"c:5"}, // ctrl-e
	},
	TutorialStep{
		title:       "Delete the current letter",
		description: "Press ctrl-d",
		expectKeys:  []string{"c:8"}, // ctrl-g
	},
	TutorialStep{
		title:       "Delete to letter to the left",
		description: "Press ctrl-h or backspace",
		expectKeys:  []string{"c:9"}, // ctrl-h or backspace
	},
	TutorialStep{
		title:       "Save",
		description: "Press ctrl-s to save.",
		expectKeys:  []string{"c:15"}, // ctrl-s
	},
	TutorialStep{
		title:       "Quit",
		description: "Press ctrl-q to quit.",
		expectKeys:  []string{"c:17"}, // ctrl-q
	},
	TutorialStep{
		title:       "Tutorial complete",
		description: "All done!",
		expectKeys:  []string{"c:32"}, // space
	},
}

// LaunchTutorial launches a short and sweet tutorial that covers at least portals and cut/paste
func LaunchTutorial(c *vt100.Canvas, e *Editor, status *StatusBar) {
	const repositionCursorAfterDrawing = false
	for _, step := range tutorialSteps {
		step.Show(c, e, status, repositionCursorAfterDrawing)
		time.Sleep(1 * time.Second)
	}
}

func (step TutorialStep) Show(c *vt100.Canvas, e *Editor, status *StatusBar, repositionCursorAfterDrawing bool) {
	canvasBox := NewCanvasBox(c)

	// Window is the background box that will be drawn in the upper right
	centerBox := NewBox()

	minWidth := 32

	centerBox.EvenLowerRightPlacement(canvasBox, minWidth)
	e.redraw = true

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(centerBox, 1, 1)

	// Get the current theme for the register box
	bt := e.NewBoxTheme()
	bt.Foreground = &e.ListTextColor
	bt.Background = &e.DebugInstructionsBackground

	lines := strings.Split(step.description, "\n")

	e.DrawBox(bt, c, centerBox)
	e.DrawTitle(bt, c, centerBox, step.title)
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
