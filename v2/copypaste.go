package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xyproto/clip"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// When pasting text, portals older than this duration will be disregarded
const maxPortalAge = 25 * time.Minute

// SetClipboardFromFile can copy the given file to the clipboard.
// The returned int is the number of bytes written.
// The returned string is the last 7 characters written to the file.
func SetClipboardFromFile(filename string, primaryClipboard bool) (int, string, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return 0, "", err
	}

	// Write to the clipboard
	if err := clip.WriteAllBytes(data, primaryClipboard); err != nil {
		return 0, "", err
	}

	contents := string(data)
	tailString := ""
	if l := len(contents); l > 7 {
		tailString = string(contents[l-8:])
	}

	return len(data), tailString, nil
}

// WriteClipboardToFile can write the contents of the clipboard to a file.
// If overwrite is true, the original file will be removed first, if it exists.
// The returned int is the number of bytes written.
// The fist returned string is the first 7 characters written to the file.
// The second returned string is the last 7 characters written to the file.
func WriteClipboardToFile(filename string, overwrite, primaryClipboard bool) (int, string, string, error) {
	// Check if the file exists first
	if files.Exists(filename) {
		if overwrite {
			if err := os.Remove(filename); err != nil {
				return 0, "", "", err
			}
		} else {
			return 0, "", "", fmt.Errorf("%s already exists", filename)
		}
	}

	// Read the clipboard
	contents, err := clip.ReadAllBytes(primaryClipboard)
	if err != nil {
		return 0, "", "", err
	}

	// Write to file
	f, err := os.Create(filename)
	if err != nil {
		return 0, "", "", err
	}
	defer f.Close()

	lenContents := len(contents)

	headString := ""
	if lenContents > 7 {
		headString = string(contents[:8])
	}

	tailString := ""
	if lenContents > 7 {
		tailString = string(contents[lenContents-8:])
	}

	n, err := f.Write(contents)
	if err != nil {
		return 0, "", "", err
	}
	return n, headString, tailString, nil
}

// Paste is called when the user presses ctrl-v, and handles portals, clipboards and also non-clipboard-based copy and paste
func (e *Editor) Paste(c *vt100.Canvas, status *StatusBar, copyLines, previousCopyLines *[]string, firstPasteAction *bool, lastCopyY, lastPasteY, lastCutY *LineIndex, prevKeyWasReturn bool) {
	if portal, err := LoadPortal(maxPortalAge); err == nil { // no error
		status.Clear(c)
		line, err := portal.PopLine(e, false) // pop the line, but don't remove it from the source file
		if err == nil {                       // success
			status.SetMessageAfterRedraw("Pasting through the portal")
			undo.Snapshot(e)
			if e.EmptyRightTrimmedLine() {
				// If the line is empty, replace with the string from the portal
				e.SetCurrentLine(line)
			} else {
				// If the line is not empty, insert the trimmed string
				e.InsertStringAndMove(c, strings.TrimSpace(line))
			}
			e.InsertLineBelow()
			e.Down(c, nil) // no status message if the end of document is reached, there should always be a new line
			e.redraw = true
			return
		}
		e.ClosePortal()
		status.SetError(err)
		status.Show(c, e)
	}

	// This may only work for the same user, and not with sudo/su

	// Try fetching the lines from the clipboard first
	var s string

	var err error
	if isDarwin() {
		s, err = pbpaste()
	} else {
		// Read the clipboard, for other platforms
		s, err = clip.ReadAll(false) // non-primary clipboard
		if err == nil && strings.TrimSpace(s) == "" {
			s, err = clip.ReadAll(true) // try the primary clipboard
		}
	}

	if err == nil { // no error

		// Make the replacements, then split the text into lines and store it in "copyLines"
		*copyLines = strings.Split(opinionatedStringReplacer.Replace(s), "\n")

		// Note that control characters are not replaced, they are just not printed.
	} else if *firstPasteAction {
		missingUtility := false

		status.Clear(c)

		if env.Has("WAYLAND_DISPLAY") && files.Which("wl-paste") == "" { // Wayland + wl-paste not found
			status.SetErrorMessage("The wl-paste utility (from wl-clipboard) is missing!")
			missingUtility = true
		} else if env.Has("DISPLAY") && files.Which("xclip") == "" { // X + xclip not found
			status.SetErrorMessage("The xclip utility is missing!")
			missingUtility = true
		} else if isDarwin() && files.Which("pbpaste") == "" { // pbcopy is missing, on macOS
			status.SetErrorMessage("The pbpaste utility is missing!")
			missingUtility = true
		}

		if missingUtility && *firstPasteAction {
			*firstPasteAction = false
			status.Show(c, e)
			return // Break instead of pasting from the internal buffer, but only the first time
		}
	} else {
		status.Clear(c)
		e.redrawCursor = true
	}

	// Now check if there is anything to paste
	if len(*copyLines) == 0 {
		return
	}

	// Now save the contents to "previousCopyLines" and check if they are the same first
	if !equalStringSlices(*copyLines, *previousCopyLines) {
		// Start with single-line paste if the contents are new
		*lastPasteY = -1
	}
	*previousCopyLines = *copyLines

	// Prepare to paste
	undo.Snapshot(e)
	y := e.DataY()

	// Forget the cut and copy line state
	*lastCutY = -1
	*lastCopyY = -1

	// Redraw after pasting
	e.redraw = true

	if *lastPasteY != y { // Single line paste
		*lastPasteY = y
		// Pressed for the first time for this line number, paste only one line

		// (*copyLines)[0] is the line to be pasted, and it exists

		if e.EmptyRightTrimmedLine() {
			// If the line is empty, use the existing indentation before pasting
			e.SetLine(y, e.LeadingWhitespace()+strings.TrimSpace((*copyLines)[0]))
		} else {
			// If the line is not empty, insert the trimmed string
			e.InsertStringAndMove(c, strings.TrimSpace((*copyLines)[0]))
		}

	} else { // Multi line paste (the rest of the lines)
		// Pressed the second time for this line number, paste multiple lines without trimming
		var (
			firstLine     = (*copyLines)[0]
			tailLines     = (*copyLines)[1:]
			tailLineCount = len(tailLines)

			// tailLines contains the lines to be pasted, and they are > 1
			// the first line is skipped since that was already pasted when ctrl-v was pressed the first time
			lastIndex = tailLineCount - 1

			// If the first line has been pasted, and return has been pressed, paste the rest of the lines differently
			skipFirstLineInsert bool
		)

		// Consider smart indentation for programming languages
		if e.ProgrammingLanguage() || e.mode == mode.Config {
			// Indent the block that is about to be pasted to the smart indentation level, if the block had no indentation
			if getLeadingWhitespace(firstLine) == "" {
				leadingWhitespace := e.LeadingWhitespace()
				// add indentation to each line
				firstLine = leadingWhitespace + firstLine
				for i := 0; i < tailLineCount; i++ {
					(*copyLines)[1+i] = leadingWhitespace + (*copyLines)[1+i]
				}
			}
		}

		if !prevKeyWasReturn {
			// Start by pasting (and overwriting) an untrimmed version of this line,
			// if the previous key was not return.
			e.SetLine(y, firstLine)
		} else if e.EmptyRightTrimmedLine() {
			skipFirstLineInsert = true
		}

		// Then paste the rest of the lines, also untrimmed
		for i, line := range tailLines {
			if i == lastIndex && len(strings.TrimSpace(line)) == 0 {
				// If the last line is blank, skip it
				break
			}
			if skipFirstLineInsert {
				skipFirstLineInsert = false
			} else {
				e.InsertLineBelow()
				e.Down(c, nil) // no status message if the end of document is reached, there should always be a new line
			}
			e.InsertStringAndMove(c, line)
		}

		if numLines := 1 + tailLineCount; numLines > 1 {
			status.SetMessageAfterRedraw(fmt.Sprintf("Pasted %d lines", numLines))
		}
	}

	// Prepare to redraw the text
	e.redrawCursor = true
	e.redraw = true
}
