package megafile

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/vt"
)

var resizeMutex sync.Mutex

// FullResetRedraw will completely reset and redraw everything, including creating a brand new Canvas struct
func (s *State) FullResetRedraw() {
	resizeMutex.Lock()
	defer resizeMutex.Unlock()

	c := s.canvas

	// Close and reset the VT terminal
	vt.Close()
	vt.Reset()
	vt.Clear()
	vt.Init()

	// Create new canvas
	newC := vt.NewCanvas()
	newC.ShowCursor()
	vt.EchoOff()

	// Assign the new canvas to the current canvas
	*c = newC.Copy()

	// Trigger a complete redraw
	s.redraw()
}

// redraw redraws the entire screen with the current state
func (s *State) redraw() {
	c := s.canvas
	o := vt.New()

	// Clear the canvas
	c.Clear()

	y := topLine

	// Redraw the header
	c.Write(5, y, s.HeaderColor, s.Background, s.Header)
	y++

	// Redraw the uptime
	const fullKernelVersion = false
	if uptimeString, err := upsieString(fullKernelVersion); err == nil {
		c.WriteTagged(5, y, s.Background, o.LightTags(uptimeString))
		y++
	}

	// Redraw directory information
	var symlinkPathMarker string
	if !s.RealPath() {
		symlinkPathMarker = ">"
	}
	c.WriteTagged(5, y, s.Background, o.LightTags(fmt.Sprintf("<yellow>%d</yellow> <gray>[</gray><green>%s</green><gray>]</gray> <magenta>%s</magenta>", s.dirIndex, s.Directories[s.dirIndex], symlinkPathMarker)))
	y++

	// Show hidden files indicator
	if s.ShowHidden {
		c.Write(5, y, vt.Default, s.Background, ".")
	} else {
		c.Write(5, y, vt.Default, s.Background, " ")
	}
	y++

	// Redraw the prompt
	prompt := ""
	if absPath, err := filepath.Abs(s.Directories[s.dirIndex]); err == nil {
		prompt = absPath
	} else {
		prompt = s.Directories[s.dirIndex]
	}
	prompt = strings.Replace(prompt, env.HomeDir(), "~", 1)
	c.Write(s.startx, s.starty, s.PromptColor, s.Background, prompt)
	s.promptLength = ulen([]rune(prompt)) + 2
	c.WriteRune(s.startx+s.promptLength-2, s.starty, s.AngleColor, s.Background, '>')
	c.WriteRune(s.startx+s.promptLength-1, s.starty, vt.Default, s.Background, ' ')

	// Redraw written text
	x := s.startx + s.promptLength
	y = s.starty
	c.Write(x, y, s.WrittenTextColor, s.Background, string(s.written))

	// Redraw file entries
	s.fileEntries = []FileEntry{}
	s.ls(s.Directories[s.dirIndex])

	// Highlight current selection if any
	if s.selectedIndex() >= 0 && s.selectedIndex() < len(s.fileEntries) {
		s.highlightSelection()
	}

	// Draw the canvas
	c.Draw()
}
