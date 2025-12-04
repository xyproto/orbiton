package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xyproto/binary"
	"github.com/xyproto/clip"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
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

	if strings.HasSuffix(filename, ".gz") {
		data, err = gUnzipData(data)
		if err != nil {
			return 0, "", err
		}
	}

	// Write to the clipboard
	if err := clip.WriteAllBytes(data, primaryClipboard); err != nil {
		return 0, "", err
	}

	tailString := ""
	if l := len(data); l > 7 {
		tailString = string(data[l-7:])
	}

	return len(data), tailString, nil
}

// emptyFile checks if the given file is empty
// if there is an error, then false is returned
func emptyFile(filename string) bool {
	fi, err := os.Stat(filename)
	if err != nil {
		return false // something went wrong, probably not an empty file
	}
	return fi.Size() == 0
}

// WriteClipboardToFile can write the contents of the clipboard to a file.
// If overwrite is true, the original file will be removed first, if it exists.
// The returned int is the number of bytes written.
// The fist returned string is the first 7 characters written to the file.
// The second returned string is the last 7 characters written to the file.
func WriteClipboardToFile(filename string, overwrite, primaryClipboard bool) (int, string, string, error) {
	// Check if the file exists first
	if files.Exists(filename) {
		if overwrite || emptyFile(filename) {
			if err := os.Remove(filename); err != nil {
				return 0, "", "", err
			}
		} else {
			return 0, "", "", fmt.Errorf("%s already exists and is not empty", filename)
		}
	}

	// Read the clipboard
	contents, err := clip.ReadAllBytes(primaryClipboard)
	if err != nil {
		return 0, "", "", err
	}

	// If it's not binary data, make sure there is a final newline
	if !binary.Data(contents) {
		if !bytes.HasSuffix(contents, []byte{'\n'}) {
			contents = append(contents, '\n')
		}
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
		headString = string(contents[:7])
	}

	tailString := ""
	if lenContents > 7 {
		tailString = string(contents[lenContents-7:])
	}

	n, err := f.Write(contents)
	if err != nil {
		return 0, "", "", err
	}
	return n, headString, tailString, nil
}

// Paste is called when the user presses ctrl-v, and handles portals, clipboards and also non-clipboard-based copy and paste
func (e *Editor) Paste(c *vt.Canvas, status *StatusBar, copyLines, previousCopyLines *[]string, firstPasteAction *bool, lastCopyY, lastPasteY, lastCutY *LineIndex, prevKeyWasReturn bool) {
	if portal, err := LoadPortal(maxPortalAge); err == nil { // no error
		line, err := portal.PopLine(e, false) // pop the line, but don't remove it from the source file
		if err == nil {                       // success
			status.ClearAll(c, false)
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
			e.redraw.Store(true)
			return
		}
		e.ClosePortal()
		status.Clear(c, false)
		status.SetError(err)
		status.Show(c, e)
	}

	// This may only work for the same user, and not with sudo/su

	// Try fetching the lines from the clipboard first
	var s string

	var err error
	if isDarwin {
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

		status.Clear(c, false)

		if env.Has("WAYLAND_DISPLAY") && files.WhichCached("wl-paste") == "" { // Wayland + wl-paste not found
			status.SetErrorMessage("The wl-paste utility (from wl-clipboard) is missing!")
			missingUtility = true
		} else if env.Has("DISPLAY") && files.WhichCached("xclip") == "" { // X + xclip not found
			status.SetErrorMessage("The xclip utility is missing!")
			missingUtility = true
		} else if isDarwin && files.WhichCached("pbpaste") == "" { // pbcopy is missing, on macOS
			status.SetErrorMessage("The pbpaste utility is missing!")
			missingUtility = true
		}

		if missingUtility && *firstPasteAction {
			*firstPasteAction = false
			status.Show(c, e)
			return // Break instead of pasting from the internal buffer, but only the first time
		}
	} else {
		status.Clear(c, true)
		e.redrawCursor.Store(true)
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
	e.redraw.Store(true)

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
		if ProgrammingLanguage(e.mode) || e.mode == mode.Config { // not mode.Ini and mode.Fstab, since those seldom have indentations
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
			if i == lastIndex && strings.TrimSpace(line) == "" {
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
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}
