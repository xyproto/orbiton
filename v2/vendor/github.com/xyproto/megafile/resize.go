package megafile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/vt"
)

var resizeMutex sync.Mutex

// FullResetRedraw will completely reset and redraw everything, including creating a brand new Canvas struct.
// Only redraws when in file browsing mode, not when an external editor/command is running.
func (s *State) FullResetRedraw() {
	if !s.browsing.Load() {
		return
	}

	resizeMutex.Lock()
	defer resizeMutex.Unlock()

	c := s.canvas

	// Close and reset the VT terminal
	vt.CloseKeepContent()
	vt.Reset()
	vt.Clear()
	vt.Init()

	// Create new canvas
	newC := vt.NewCanvas()
	newC.ShowCursor()
	vt.EchoOff()

	// Assign the new canvas to the current canvas
	*c = newC.Copy()

	// Create another new canvas to ensure the terminal size is correct after resize
	newC = vt.NewCanvas()
	newC.ShowCursor()
	vt.EchoOff()

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
	if uptimeString, err := UpsieString(fullKernelVersion); err == nil {
		if envNoColor {
			c.WriteTagged(5, y, s.Background, uptimeString)
		} else {
			c.WriteTagged(5, y, s.Background, o.LightTags(uptimeString))
		}
		y++
	}

	// Redraw directory information
	var symlinkPathMarker string
	if !s.RealPath() {
		symlinkPathMarker = ">"
	}
	if envNoColor {
		c.WriteTagged(5, y, s.Background, fmt.Sprintf("%d [%s] %s", s.dirIndex, s.Directories[s.dirIndex], symlinkPathMarker))
	} else {
		c.WriteTagged(5, y, s.Background, o.LightTags(fmt.Sprintf("<yellow>%d</yellow> <gray>[</gray><green>%s</green><gray>]</gray> <magenta>%s</magenta>", s.dirIndex, s.Directories[s.dirIndex], symlinkPathMarker)))
	}
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

// startResizeHandler starts a goroutine that listens for terminal resize
// signals and triggers a full redraw. Only one handler is active at a time.
func (s *State) startResizeHandler() {
	s.stopResizeHandler()

	sigChan := make(chan os.Signal, 1)
	done := make(chan struct{})
	s.resizeChan = sigChan
	s.resizeCancel = func() {
		ResetResizeSignal()
		close(done)
	}

	SetupResizeSignal(sigChan)

	go func() {
		for {
			select {
			case <-sigChan:
				s.FullResetRedraw()
				time.Sleep(150 * time.Millisecond)
				s.FullResetRedraw()
			case <-done:
				return
			}
		}
	}()
}

// stopResizeHandler stops the current resize signal handler goroutine.
func (s *State) stopResizeHandler() {
	if s.resizeCancel != nil {
		s.resizeCancel()
		s.resizeCancel = nil
		s.resizeChan = nil
	}
}
