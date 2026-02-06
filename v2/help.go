package main

import (
	"fmt"
	"strings"

	"github.com/xyproto/vt"
)

const (
	// macOS-style pico/nano help text
	nanoHelpString1 = " ^G Get Help  ^O Write Out  ^R Read File  ^Y Prev Pg  ^K Cut Text    ^C Cur Pos  "
	nanoHelpString2 = " ^X Exit      ^J Justify    ^W Where Is   ^V Next Pg  ^U UnCut Text  ^T To Spell "
)

var (
	// NOTE: The DrawQuickHelp function requires the wording of "Disable this overview" to stay the same
	quickHelpText = `Save                   ctrl-s
Quit                   ctrl-q
Main menu              ctrl-o
Overview of hotkeys    ctrl-l and then /
Launch tutorial        ctrl-l and then ?
Disable this overview  ctrl-l and then !`

	ollamaHelpText = "Ollama"

	usageText = `Hotkeys

ctrl-s      to save
ctrl-q      to quit
ctrl-o      to open the command menu
ctrl-r      to open a portal so that text can be pasted into another file with ctrl-v
ctrl-space  to compile programs or export adoc/sdoc as a man page
            toggle checkboxes in Markdown, or double press to render the file as HTML
ctrl-w      for Zig, Rust, V and Go, format with the "... fmt" command
            for C++, format the current file with "clang-format"
            for HTML, format the file with "tidy", for Python: "black"
            for Markdown, toggle checkboxes or re-format tables
            for git interactive rebases, cycle the rebase keywords
ctrl-g      jump to include, definition (experimental), back or toggle the status bar
ctrl-_      insert a symbol by typing in a two letter ViM-style digraph
            see https://raw.githubusercontent.com/xyproto/digraph/main/digraphs.txt
ctrl-a      go to start of line, then start of text and then the previous line
ctrl-e      go to end of line and then the next line
ctrl-n      to scroll down 10 lines or go to the next match if a search is active
            insert a column when in the Markdown table editor
            go to next match when searching, or next typo when spellchecking
            jump to a matching parenthesis or bracket if the arrow keys were just used
ctrl-p      to scroll up 10 lines or go to the previous match
            remove an empty column when in the Markdown table editor
            jump to a matching parenthesis or bracket if the arrow keys were just used
ctrl-k      to delete characters to the end of the line, then delete the line
ctrl-j      to join lines
ctrl-d      to delete a single character
ctrl-t      for C and C++, toggle between the header and implementation,
            for Markdown, toggle checkboxes or launch the table editor
            for Agda, insert a symbol,
            for the rest, record and then play back a macro
ctrl-c      to copy the current line, press twice to copy the current block
            press thrice to copy the current function
ctrl-v      to paste one line, press twice to paste the rest
ctrl-x      to cut the current line, press twice to cut the current block
ctrl-b      to jump back after having jumped to a definition or include
            to toggle a bookmark for the current line, or jump to a bookmark
            to toggle a breakpoint if in debug mode
ctrl-u      to undo (ctrl-z is also possible, but may background the application)
ctrl-l      to jump to a specific line or letter (press return to jump to the top or bottom)
ctrl-f      to find text. To search and replace, press Tab instead of Return.
            to spellcheck, search for "t", then press ctrl-a to add and ctrl-i to ignore
ctrl-\      to toggle single-line comments for a block of code
ctrl-~      insert the current date and time
esc         to redraw the screen, clear the last search and clear the current macro

Set NO_COLOR=1 to disable colors.

Flags:
  -c, --copy FILENAME            Copy the given file into the clipboard.
  -p, --paste FILENAME           Paste the contents of the clipboard into the given file.
                                 Combine with -f to overwrite the file.
  -f, --force                    Ignore file locks when opening files.
  -l, --last-command             Output the last used build/format/export command.
  -e, --clear-locks              Clear all file locks and close all portals.
  -m, --monitor FILENAME         Monitor the given file for changes, and open it as read-only.
  -o, --ollama                   Use $OLLAMA$
                                 to explain the function under the cursor.
  -r, --release                  Build with release instead of debug mode whenever applicable.
  -x, --noapprox                 Disable approximate filename matching.
  -n, --no-cache                 Avoid writing the location history, search history, highscore,
                                 compilation and format command to ` + cacheDirForDoc + `.
  -d, --create-dir               When opening a new file, create directories as needed.
  -s, --digraphs                 List all possible digraphs.
  -t, --list                     List the given file using the red/black theme and quit.
  -b, --bat                      List the given file using bat, if it exists in the PATH.
                                 This can be useful when used with together with -c or -p.
  -i, --input-file FILENAME      Used as stdin when running programs with ctrl-space.
                                 The default filename is input.txt. Handy for Advent of Code.
  -a, --nano                     Emulate Pico/Nano.
  -q, --quick-help               Always display the quick help pane at start.
  -z, --no-quick-help            Never display the quick help pane at start.
  -k, --slowkey                  Use a longer ESC timeout for slow terminals.
  -g, --glob GLOB                Search for and open the first filename that matches the substring.
  -h, --help                     Display this usage information.
  -y, --esc                      Just pressing Esc will exit the program.
  -v, --version                  Display the current version.

See the man page for more information.

`
)

// Usage prints the text that appears when the --help flag is passed
func Usage() {
	fmt.Println(versionString + " - simple and limited text editor")
	fmt.Print(strings.Replace(usageText, "$OLLAMA$", ollamaHelpText, 1))
}

// DrawNanoHelp will draw a help box for nano hotkeys in the center
func (e *Editor) DrawNanoHelp(c *vt.Canvas, repositionCursorAfterDrawing bool) {
	const (
		maxLines     = 30
		title        = "Orbiton Nano Mode"
		nanoHelpText = `The Orbiton Nano mode is designed to emulate the core functionality
and relative easy-of-use of the UW Pico and GNU Nano editors.

Keyboard hotkeys:

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
            after pressing ctrl-t to find typos, add a word to the dictionary
ctrl-e    - go to the end of line and then to the next line
ctrl-n    - go to the next line, or to the next match if a search is active
ctrl-p    - go to the previous line, or to the next match if a search is active
ctrl-d    - delete a single character
ctrl-f    - move cursor one position forward
ctrl-b    - move cursor one position back
ctrl-l    - refresh the current screen
`
	)

	var (
		minWidth        = 40
		backgroundColor = e.DebugRunningBackground
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
	const marginY = 2
	centerBox.FillWithMargins(canvasBox, marginX, marginY)
	centerBox.Y--

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(centerBox, 2, 2)

	// Get the current theme for the stdout box
	bt := e.NewBoxTheme()
	bt.Background = &backgroundColor
	bt.UpperEdge = bt.LowerEdge

	e.DrawBox(bt, c, centerBox)

	e.DrawTitle(bt, c, centerBox, title, true)

	e.DrawList(bt, c, listBox, lines, -1)

	// Blit
	c.HideCursorAndDraw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}

// DrawHotkeyOverview shows an overview of Orbiton hotkeys
func (e *Editor) DrawHotkeyOverview(tty *vt.TTY, c *vt.Canvas, status *StatusBar, repositionCursorAfterDrawing bool) {
	const title = "Hotkey overview"

	// Extracting hotkey information from usageText
	startIndex := strings.Index(usageText, "Hotkeys")
	if startIndex == -1 {
		return
	}
	endIndex := strings.Index(usageText, "Set NO_COLOR=1")
	if endIndex == -1 {
		return
	}
	hotkeyInfo := usageText[startIndex:endIndex]

	// Split the hotkeyInfo into lines
	hotkeyLines := strings.Split(hotkeyInfo, "\n")

	// Calculate the box width and height as 80% of the canvas height
	pageWidth := int(float64(c.Width()) * 0.8)
	pageHeight := int(float64(c.Height()) * 0.8)

	// Create pages of text
	var pages []Page
	for i := 0; i < len(hotkeyLines); i += pageHeight {
		end := i + pageHeight
		if end > len(hotkeyLines) {
			end = len(hotkeyLines)
		}
		pages = append(pages, Page{Lines: hotkeyLines[i:end]})
	}

	// TODO: Clean up the following block of code, remove reduntant lines!
	canvasBox := NewCanvasBox(c)
	centerBox := NewBox()
	const marginX = 5
	const marginY = 2
	centerBox.FillWithMargins(canvasBox, marginX, marginY)
	centerBox.Y--
	centerBox.W = pageWidth
	centerBox.H = pageHeight + 6
	scrollableTextBox := NewScrollableTextBox(pages)
	scrollableTextBox.FillWithMargins(centerBox, 4, 4)
	boxTheme := e.NewBoxTheme()
	boxTheme.Foreground = &e.TableColor
	boxTheme.Background = &e.NanoHelpBackground
	surroundingBox := *(scrollableTextBox.Box)
	surroundingBox.X -= 2
	surroundingBox.Y -= 2
	surroundingBox.W += 2
	surroundingBox.H += 4

	for {
		// Draw the current page
		e.DrawBox(boxTheme, c, &surroundingBox)
		e.DrawTitle(boxTheme, c, &surroundingBox, title, true)
		if len(pages) > 1 {
			status.SetMessage("Press Space to view the next page. Press q or Esc to close.")
			status.Show(c, e)
		} else {
			status.SetMessage("Press Esc or q to close.")
			status.Show(c, e)
		}
		e.DrawScrollableText(boxTheme, c, scrollableTextBox)
		c.HideCursorAndDraw()

		// Wait for a keypress
		key := tty.ReadStringEvent()
		switch key {
		case " ": // Space key to go to next page
			scrollableTextBox.NextPage()
		case "c:13", "c:17", "c:27", "q": // return, ctrl-q, esc or q
			goto endOfLoop
		case "↓", "j", "c:14": // down, j or ctrl-n
			scrollableTextBox.NextPage()
		case "↑", "k", "c:16": // up, k or ctrl-p
			scrollableTextBox.PrevPage()
		}
	}

endOfLoop:

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}
