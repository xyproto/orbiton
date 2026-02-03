package main

import (
	"fmt"
	"time"

	"github.com/xyproto/vt"
)

var pacmanNoColor = []string{
	"| > · · |",
	"|  >· · |",
	"|   > · |",
	"|    >· |",
	"|     > |",
	"|      -|",
	"| · · < |",
	"| · ·<  |",
	"| · <   |",
	"| ·<    |",
	"| <     |",
	"|-· · · |",
}

var pacmanColor = []string{
	"<red>| <yellow>C<blue> · ·</blue> <red>|<off>",
	"<red>| <blue> <yellow>⊂<blue>· · <red>|<off>",
	"<red>| <blue>  <yellow>⊂<blue> · <red>|<off>",
	"<red>| <blue>   <yellow>⊂<blue>· <red>|<off>",
	"<red>| <blue>    <yellow>⊂ <red>|<off>",
	"<red>| <blue>     <yellow>○<red>|<off>",
	"<red>| <blue>· · <yellow>Ɔ <red>|<off>",
	"<red>| <blue>· ·<yellow>⊃<blue>  <red>|<off>",
	"<red>| <blue>· <yellow>⊃ <blue>  <red>|<off>",
	"<red>| <blue>·<yellow>⊃<blue>    <red>|<off>",
	"<red>| <yellow>⊃ <blue>    <red>|<off>",
	"<red>|<yellow>○<blue>· · · <red>|<off>",
}

var spinnerASCII = []string{
	"-",
	"\\",
	"|",
	"/",
}

// Spinner waits a bit, then displays a spinner together with the given message string (msg).
// If the spinner is aborted, the qmsg string is displayed.
// Returns a quit channel (chan bool).
// The spinner is shown asynchronously.
// "true" must be sent to the quit channel once whatever operating that the spinner is spinning for is completed.
func (e *Editor) Spinner(c *vt.Canvas, tty *vt.TTY, umsg, qmsg string, startIn time.Duration, textColor vt.AttributeColor, cursorAfterText bool) chan bool {
	quitChan := make(chan bool)
	go func() {
		// Divide the startIn time into 5, then wait while listening to the quitChan
		// If the quitChan does not receive anything by then, show the spinner
		const N = 50
		for i := 0; i < N; i++ {
			// Check if we should quit or wait
			select {
			case <-quitChan:
				return
			default:
				// Wait a tiny bit
				time.Sleep(startIn / N)
			}
		}

		// If c or tty are nil, use the silent spinner
		if (c == nil) || (tty == nil) {
			// Wait for a true on the quit channel, then return
			<-quitChan
			return
		}

		// Get the terminal codes for coloring the given user message
		msg := textColor.Get(umsg)
		if envVT100 {
			msg = umsg
		}

		var x, y uint
		if cursorAfterText {
			x, y = e.GetXYAfterText()
			// Store the position after the message
			//x += ulen(msg) + 1
			// Move the cursor there
			//vt.SetXY(x, y)
		} else {
			// Find a good start location
			x = uint(int(c.Width()) / 7)
			y = uint(int(c.Height()) / 7)
			// Move the cursor there
			vt.SetXY(x, y)
			// Store the position after the message
			x += ulen(msg) + 1
		}

		// Write a message
		fmt.Print(msg)

		// Prepare to output colored text
		var (
			to               = vt.New()
			counter          uint
			spinnerAnimation []string
		)

		// Hide the cursor
		vt.ShowCursor(false)
		defer vt.ShowCursor(true)

		// Echo off
		vt.EchoOff()

		if envVT100 {
			spinnerAnimation = spinnerASCII
		} else if envNoColor {
			spinnerAnimation = pacmanNoColor
		} else {
			spinnerAnimation = pacmanColor
		}

		// Start the spinner
		for {
			select {
			case <-quitChan:
				return
			default:
				vt.SetXY(x, y)
				// Iterate over the spinner frames as the counter increases
				to.Print(spinnerAnimation[counter%ulen(spinnerAnimation)])
				counter++
				// Wait for a key press (also sleeps just a bit)
				switch tty.KeyRaw() {
				case 27, 113, 17, 3: // esc, q, ctrl-q or ctrl-c
					quitMessage(tty, qmsg)
				}
				time.Sleep(32 * time.Millisecond) // for a smoother animation
			}
		}
	}()
	return quitChan
}
