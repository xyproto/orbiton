package main

import (
	"fmt"
	"strings"

	"github.com/xyproto/vt100"
)

const (
	// nano on macOS (a symlink from "nano" to "pico")
	nanoHelpString1 = "^G Get Help  ^O Write Out  ^R Read File  ^Y Prev Pg  ^K Cut Text    ^C Cur Pos"
	nanoHelpString2 = "^X Exit      ^J Justify    ^W Where Is   ^V Next Pg  ^U UnCut Text  ^T To Spell"

	// GNU Nano
	//nanoHelpString1 = "^G Help  ^O Write Out  ^W Where Is  ^K Cut    ^T Execute  ^C Location    M-U Undo  M-A Set Mark  M-] To Bracket  M-Q Previous"
	//nanoHelpString2 = "^X Exit  ^R Read File  ^\\ Replace  ^U Paste  ^J Justify  ^/ Go To Line  M-E Redo  M-6 Copy      ^Q Where Was    M-W Next"

)

var (
	usageText = `Hotkeys

ctrl-s      to save
ctrl-q      to quit
ctrl-o      to open the command menu
ctrl-r      to open a portal so that text can be pasted into another file with ctrl-v
ctrl-space  to compile programs or export adoc/sdoc as a man page
            double press to render Markdown as HTML
ctrl-w      for Zig, Rust, V and Go, format with the "... fmt" command
            for C++, format the current file with "clang-format"
            for HTML, format the file with "tidy", for Python: "autopep8"
            for Markdown, toggle checkboxes or re-format tables
            for git interactive rebases, cycle the rebase keywords
ctrl-g      to display simple help 2 times, then toggle the status bar
            can jump to definition (experimental feature)
ctrl-_      jump to a matching parenthesis or bracket if on one,
            otherwise insert a symbol by typing in a two letter ViM-style digraph
            see https://raw.githubusercontent.com/xyproto/digraph/main/digraphs.txt
ctrl-a      go to start of line, then start of text and then the previous line
ctrl-e      go to end of line and then the next line
ctrl-n      to scroll down 10 lines or go to the next match if a search is active
            insert a column when in the Markdown table editor
ctrl-p      to scroll up 10 lines or go to the previous match
            remove an empty column when in the Markdown table editor
ctrl-k      to delete characters to the end of the line, then delete the line
ctrl-j      to join lines
ctrl-d      to delete a single character
ctrl-t      for C and C++, toggle between the header and implementation,
            for Markdown, launch the Markdown table editor if the cursor is on a table
            for Agda, insert a symbol,
            for the rest, record and then play back a macro
ctrl-c      to copy the current line, press twice to copy the current block
ctrl-v      to paste one line, press twice to paste the rest
ctrl-x      to cut the current line, press twice to cut the current block
ctrl-b      to jump back after having jumped to a definition
            to toggle a bookmark for the current line, or jump to a bookmark
            to toggle a breakpoint if in debug mode
ctrl-u      to undo (ctrl-z is also possible, but may background the application)
ctrl-l      to jump to a specific line or letter (press return to jump to the top or bottom)
ctrl-f      to find a string, press Tab after the text to search and replace
ctrl-\      to toggle single-line comments for a block of code
ctrl-~      to jump to matching parenthesis
esc         to redraw the screen and clear the last search

Set NO_COLOR=1 to disable colors.

Flags:
  -c FILENAME                - Copy the given file into the clipboard.
  -f                         - Ignore file locks when opening files.
  -l                         - Output the last used build/format/export command.
  -m FILENAME                - Monitor the given file for changes, and open it as read-only.
  -n                         - Avoid writing the location history, search history, highscore,
                               compilation and format command to ` + cacheDirForDoc + `.
  -p FILENAME                - Paste the contents of the clipboard into the given file.
                               Combine with -f to overwrite the file.
  -r                         - Clear all file locks.
  --version                  - Display the current version.

See the man page for more information.

`
)

// Usage prints the text that appears when the --help flag is passed
func Usage() {
	fmt.Println(versionString + " - simple and limited text editor")
	fmt.Print(usageText)
}

// HelpMessage tries to show a friendly help message to the user.
func (e *Editor) HelpMessage(c *vt100.Canvas, status *StatusBar) {
	status.ClearAll(c)
	status.SetMessage("Press ctrl-q to quit or ctrl-o to show the menu. Use the --help flag for more help.")
	status.Show(c, e)
}

// DrawNanoHelp will draw a help box for nano hotkeys in the center
func (e *Editor) DrawNanoHelp(c *vt100.Canvas, repositionCursorAfterDrawing bool) {
	const (
		maxLines     = 30
		title        = "Orbiton Nano Mode"
		nanoHelpText = `The Orbiton Nano mode is designed to emulate the core functionality and relative easy-of-use of the UW Pico and GNU Nano editors.

Keyboard shortcuts:

ctrl-g    - display this help
ctrl-o    - save this file as a different filename ("Write Out")
ctrl-r    - insert a file ("Read File")
ctrl-y    - page up ("Prev Pg")
ctrl-v    - page down ("Next Pg")
ctrl-k    - cut this line ("Cut Text")
ctrl-c    - display brief cursor location information ("Cur Pos")
ctrl-x    - quit without saving ("Exit)
ctrl-j    - join this block of text ("Justify")
ctrl-w    - search ("Where Is")
ctrl-q    - search backwards
ctrl-u    - paste ("UnCut Text")
ctrl-t    - jump to the next misspelled English word ("To Spell")
ctrl-/    - go to line
ctrl-s    - save
ctrl-a    - go to the start of the line, start of the text or one line up
ctrl-e    - go to the end of line and then to the next line
ctrl-n    - move to the next line, or go to the next match if a search is active
ctrl-p    - move to the previous line, or go to the next match if a search is active
ctrl-d    - delete a single character
ctrl-f    - move cursor one position forward
ctrl-b    - move cursor one position back
ctrl-l    - refresh the current screen
`
	)

	var (
		minWidth        = 40
		backgroundColor = vt100.BackgroundGray
	)

	// Get the last maxLine lines, and create a string slice
	lines := strings.Split(nanoHelpText, "\n")
	if l := len(lines); l > maxLines {
		lines = lines[l-maxLines:]
	}
	for _, line := range lines {
		if len(line) > minWidth {
			minWidth = len(line) + 5
		}
	}

	// First create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	centerBox := NewBox()

	const marginX = 5
	const marginY = 5
	centerBox.FillWithMargins(canvasBox, marginX, marginY)

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(centerBox, 2, 2)

	// Get the current theme for the stdout box
	bt := e.NewBoxTheme()
	bt.Background = &backgroundColor
	bt.UpperEdge = bt.LowerEdge

	e.DrawBox(bt, c, centerBox)

	e.DrawTitle(bt, c, centerBox, title)

	e.DrawList(bt, c, listBox, lines, -1)

	// Blit
	c.Draw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}
}
