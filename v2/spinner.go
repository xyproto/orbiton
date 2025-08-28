package main

import (
	"fmt"
	"time"
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

// Spinner waits a bit, then displays a spinner together with the given message string (msg).
// If the spinner is aborted, the qmsg string is displayed.
// Returns a quit channel (chan bool).
// The spinner is shown asynchronously.
// "true" must be sent to the quit channel once whatever operating that the spinner is spinning for is completed.
func Spinner(c *Canvas, tty *TTY, umsg, qmsg string, startIn time.Duration, textColor AttributeColor) chan bool {
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

		var (
			// Find a good start location
			x = uint(int(c.Width()) / 7)
			y = uint(int(c.Height()) / 7)

			// Get the terminal codes for coloring the given user message
			msg = textColor.Get(umsg)
		)

		// Move the cursor there and write a message
		SetXY(x, y)
		fmt.Print(msg)

		// Store the position after the message
		x += uint(len(msg)) + 1

		// Prepare to output colored text
		var (
			to               = New()
			counter          uint
			spinnerAnimation []string
		)

		// Hide the cursor
		ShowCursor(false)
		defer ShowCursor(true)

		// Echo off
		EchoOff()

		if envNoColor {
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
				SetXY(x, y)
				// Iterate over the 12 different ASCII images as the counter increases
				to.Print(spinnerAnimation[counter%12])
				counter++
				// Wait for a key press (also sleeps just a bit)
				switch tty.Key() {
				case 27, 113, 17, 3: // esc, q, ctrl-q or ctrl-c
					quitMessage(tty, qmsg)
				}
			}
		}
	}()
	return quitChan
}
