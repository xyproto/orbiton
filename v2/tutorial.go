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
	feedback    string
}

type Tutorial []TutorialStep

var tutorialSteps = Tutorial{
	TutorialStep{
		title:       "Save",
		description: "Press ctrl-s to save the current file.",
		expectKeys:  []string{"c:19"}, // ctrl-s
		feedback:    "Great!",
	},
	TutorialStep{
		title:       "Go to start of line",
		description: "Press ctrl-a to go to the start of the line.",
		expectKeys:  []string{"c:1"}, // ctrl-a
		feedback:    "Nice.",
	},
	TutorialStep{
		title:       "Quit",
		description: "Press ctrl-q to quit.",
		expectKeys:  []string{"c:17"}, // ctrl-q
		feedback:    "Allright.",
	},
	TutorialStep{
		title:       "Tutorial complete",
		description: "All done!",
		expectKeys:  []string{"c:32"}, // space
		feedback:    "All done!",
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
