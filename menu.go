package main

import (
	"errors"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/xyproto/vt100"
)

var errNoLetter = errors.New("no letter")

// getLetter returns the Nth letter in a given string, as lowercase. Ignores numbers, special characters, whitespace etc.
func getLetter(s string, pos int) (rune, error) {
	counter := 0
	for _, letter := range s {
		if unicode.IsLetter(letter) {
			if counter == pos {
				return unicode.ToLower(letter), nil
			}
			counter++
		}
	}
	return rune(0), errNoLetter
}

// selectionLetterForChoices loops through the choices and finds a unique letter (possibly the first one in the choice text),
// that can be used later for selecting that choice. The returned map maps from the letter to the choice index.
// choices where no appropriate letter were found are skipped. q is ignored, because it's reserved for "quit".
// Only the first 5 letters of each choice are considered.
func selectionLettersForChoices(choices []string) map[rune]uint {
	// TODO: Find the next item starting with this letter, with wraparound
	// Select the item that starts with this letter, if possible. Try the first, then the second, etc, up to 7
	selectionLetterMap := make(map[rune]uint)
	for index, choice := range choices {
		for pos := 0; pos < 7; pos++ {
			letter, err := getLetter(choice, pos)
			if err == nil {
				_, exists := selectionLetterMap[letter]
				// If the letter is not already stored in the keymap, and it's not q
				if !exists && (letter != 'q') {
					selectionLetterMap[letter] = uint(index)
					// Found a letter for this choice, move on
					break
				}
			}
		}
		// Did not find a letter for this choice, move on
	}
	return selectionLetterMap
}

// Menu starts a loop where keypresses are handled. When a choice is made, a number is returned.
// -1 is "no choice", 0 and up is which choice were selected.
// initialMenuIndex is the choice that should be highlighted when displaying the choices.
func (e *Editor) Menu(status *StatusBar, tty *vt100.TTY, title string, choices []string, titleColor, arrowColor, textColor, highlightColor, selectedColor vt100.AttributeColor, initialMenuIndex int, extraDashes bool) int {

	// Clear the existing handler
	signal.Reset(syscall.SIGWINCH)

	var (
		selectionLetterMap = selectionLettersForChoices(choices)
		selectedDelay      = 100 * time.Millisecond
		c                  = vt100.NewCanvas()
		resizeMut          = &sync.RWMutex{} // used when the terminal is resized
		menu               = NewMenuWidget(title, choices, titleColor, arrowColor, textColor, highlightColor, selectedColor, c.W(), c.H(), extraDashes, selectionLetterMap)
		sigChan            = make(chan os.Signal, 1)
		running            = true
		changed            = true
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
				menu.Draw(c)
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
	menu.SelectIndex(uint(initialMenuIndex))

	for running {

		// Draw elements in their new positions

		if changed {
			resizeMut.RLock()
			menu.Draw(c)
			resizeMut.RUnlock()

			// Update the canvas
			c.Draw()
		}

		// Handle events
		key := tty.String()
		switch key {
		case "↑", "←", "c:16": // Up, left or ctrl-p
			resizeMut.Lock()
			menu.Up(c)
			changed = true
			resizeMut.Unlock()
		case "↓", "→", "c:14": // Down, right or ctrl-n
			resizeMut.Lock()
			menu.Down(c)
			changed = true
			resizeMut.Unlock()
		case "c:1": // Top, ctrl-a
			resizeMut.Lock()
			menu.SelectFirst()
			changed = true
			resizeMut.Unlock()
		case "c:5": // Bottom, ctrl-e
			resizeMut.Lock()
			menu.SelectLast()
			changed = true
			resizeMut.Unlock()
		case "c:27", "q", "c:3", "c:17", "c:15": // ESC, q, ctrl-c, ctrl-q or ctrl-o
			running = false
			changed = true
		case " ", "c:13": // Space or Return
			resizeMut.Lock()
			menu.Select()
			resizeMut.Unlock()
			running = false
			changed = true
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // 0 .. 9
			number, err := strconv.Atoi(key)
			if err != nil {
				break
			}
			resizeMut.Lock()
			menu.SelectIndex(uint(number))
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
			// Choose the index for the letter that was pressed and found in the keymap, if found
			for letter, index := range selectionLetterMap {
				if letter == r {
					resizeMut.Lock()
					menu.SelectIndex(uint(index))
					changed = true
					resizeMut.Unlock()
				}
			}
		}

		// If the menu was changed, draw the canvas
		if changed {
			c.Draw()
		}

	}

	if menu.Selected() >= 0 {
		// Draw the selected item in a different color for a very short while
		resizeMut.Lock()
		menu.SelectDraw(c)
		resizeMut.Unlock()
		c.Draw()
		time.Sleep(selectedDelay)
	}

	// Clear the existing handler
	signal.Reset(syscall.SIGWINCH)

	// Restore the resize handler
	e.SetUpResizeHandler(c, tty, status)

	return menu.Selected()
}
