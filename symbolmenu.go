package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/xyproto/vt100"
)

// SymbolMenu starts a loop where keypresses are handled. When a choice is made, a number is returned.
// x and y are returned. -1,-1 is "no choice", 0,0 is the top left index.
func (e *Editor) SymbolMenu(status *StatusBar, tty *vt100.TTY, title string, choices [][]string, titleColor, arrowColor, textColor, highlightColor, selectedColor vt100.AttributeColor) (int, int) {

	// Clear the existing handler
	signal.Reset(syscall.SIGWINCH)

	var (
		selectedDelay = 100 * time.Millisecond
		c             = vt100.NewCanvas()
		resizeMut     = &sync.RWMutex{} // used when the terminal is resized
		symbolMenu    = NewSymbolWidget(title, choices, titleColor, arrowColor, textColor, highlightColor, selectedColor, c.W(), c.H())
		sigChan       = make(chan os.Signal, 1)
		running       = true
		changed       = true
	)

	// Set up a new resize handler
	signal.Notify(sigChan, syscall.SIGWINCH)

	go func() {
		for range sigChan {
			resizeMut.Lock()
			// Create a new canvas, with the new size
			nc := c.Resized()
			if nc != nil {
				vt100.Clear()
				c = nc
				symbolMenu.Draw(c)
				c.Redraw()
				changed = true
			}

			// Inform all elements that the terminal was resized
			resizeMut.Unlock()
		}
	}()

	vt100.Clear()
	vt100.Reset()
	c.Redraw()

	// Set the initial menu index
	symbolMenu.SelectIndex(0, 0)

	var collectedString string

	for running {

		// Draw elements in their new positions

		if changed {
			resizeMut.RLock()
			symbolMenu.Draw(c)
			resizeMut.RUnlock()

			// Update the canvas
			c.Draw()
		}

		// Handle events
		key := tty.String()
		switch key {
		case "↑", "c:16": // Up or ctrl-p
			resizeMut.Lock()
			symbolMenu.Up(c)
			changed = true
			resizeMut.Unlock()
		case "←": // Left
			resizeMut.Lock()
			symbolMenu.Left(c)
			changed = true
			resizeMut.Unlock()
		case "↓", "c:14": // Down, right or ctrl-n
			resizeMut.Lock()
			symbolMenu.Down(c)
			changed = true
			resizeMut.Unlock()
		case "→": // Down, right or ctrl-n
			resizeMut.Lock()
			symbolMenu.Right(c)
			changed = true
			resizeMut.Unlock()
		case "c:1": // Top, ctrl-a
			resizeMut.Lock()
			symbolMenu.SelectFirst()
			changed = true
			resizeMut.Unlock()
		case "c:5": // Bottom, ctrl-e
			resizeMut.Lock()
			symbolMenu.SelectLast()
			changed = true
			resizeMut.Unlock()
		case "c:27", "q", "c:3", "c:17", "c:15": // ESC, q, ctrl-c, ctrl-q or ctrl-o
			running = false
			changed = true
		case " ", "c:13": // Space or Return
			resizeMut.Lock()
			symbolMenu.Select()
			resizeMut.Unlock()
			running = false
			changed = true
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // 0 .. 9
			number, err := strconv.Atoi(key)
			if err != nil {
				break
			}
			resizeMut.Lock()
			symbolMenu.SelectIndex(0, uint(number))
			changed = true
			resizeMut.Unlock()
		default:
			if len([]rune(key)) == 0 {
				// this happens if pgup or pgdn is pressed
				break
			}
			// Check if the key matches the first letter (A-Z, a-z) in the choices
			r := []rune(key)[0]
			if !(65 <= r && r <= 90) && !(97 <= r && r <= 122) {
				break
			}

			// 			for _, runeAndPosition := range selectionLetterMap {
			// 				if r == runeAndPosition.r {
			// 					resizeMut.Lock()
			// 					symbolMenu.SelectIndex(runeAndPosition.pos)
			// 					changed = true
			// 					resizeMut.Unlock()
			// 				}
			// 			}

		}

		// Konami code collector
		if len([]rune(key)) == 1 {
			collectedString += key
			// pop a letter in front of the collected string if it's too long
			if len(collectedString) > len(konami) {
				runes := []rune(collectedString)
				collectedString = string(runes[1:])
			}
		}
		// Was it the konami code?
		if collectedString == konami {
			collectedString = ""
			// Start the game
			if ctrlq, err := Game(); err != nil {
				// This should never happen
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			} else if ctrlq {
				// ctrl-q was pressed, quit entirely
				running = false
				e.quit = true
			} else {
				// The game ended, return from the menu
				running = false
				changed = true
			}
		}

		// If the menu was changed, draw the canvas
		if changed {
			c.Draw()
		}

	}

	if selectedX, selectedY := symbolMenu.Selected(); selectedX >= 0 && selectedY >= 0 {
		// Draw the selected item in a different color for a very short while
		resizeMut.Lock()
		symbolMenu.SelectDraw(c)
		resizeMut.Unlock()
		c.Draw()
		time.Sleep(selectedDelay)
	}

	// Clear the existing handler
	signal.Reset(syscall.SIGWINCH)

	// Restore the resize handler
	e.SetUpResizeHandler(c, tty, status)

	return symbolMenu.Selected()
}
