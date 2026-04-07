package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// Save will try to save the current editor contents to file.
// It needs a canvas in case trailing spaces are stripped and the cursor needs to move to the end.
func (e *Editor) Save(c *vt.Canvas, tty *vt.TTY) error {
	return e.SaveAs(c, tty, e.filename)
}

// SaveAs will try to save the current editor contents to given file.
// It needs a canvas in case trailing spaces are stripped and the cursor needs to move to the end.
func (e *Editor) SaveAs(c *vt.Canvas, tty *vt.TTY, filename string) error {

	if e.monitorAndReadOnly {
		return errors.New("file is read-only")
	}

	var (
		bookmark = e.pos.Copy() // Save the current position
		changed  bool
		shebang  bool
		data     []byte
	)

	quitMut.Lock()
	defer quitMut.Unlock()

	if e.createDirectoriesIfMissing {
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return err
		}
	}

	if e.binaryFile {
		data = []byte(e.String())
	} else {
		// Strip trailing spaces on all lines
		l := e.Len()
		for i := range l {
			if e.TrimRight(LineIndex(i)) {
				changed = true
			}
		}

		// Trim away trailing whitespace
		s := trimRightSpace(e.String())

		// Make additional replacements, and add a final newline
		s = opinionatedStringReplacer.Replace(s) + "\n"

		// TODO: Auto-detect tabs/spaces instead of per-language assumptions
		if e.mode.Spaces() {
			// NOTE: This is a hack, that can only replace 10 levels deep.
			for level := 10; level > 0; level-- {
				fromString := "\n" + strings.Repeat("\t", level)
				toString := "\n" + strings.Repeat(" ", level*e.indentation.PerTab)
				s = strings.ReplaceAll(s, fromString, toString)
			}
		} else if e.mode == mode.Make || e.mode == mode.Just {
			var (
				level                          int
				fromString, toString, prevLine string
				lines                          = strings.Split(s, "\n")
			)
		NEXTLINE:
			for i, line := range lines {
				// NOTE: This is a hack, that can only replace 10 levels deep.
				for level = 10; level > 0; level-- {
					if strings.HasPrefix(prevLine, "  ") && len(prevLine) > 2 && prevLine[2] != ' ' {
						// make no replacements for this case, where the previous line has a 2-space indentation
						continue NEXTLINE
					}
					fromString = "\n" + strings.Repeat(" ", level*e.indentation.PerTab)
					toString = "\n" + strings.Repeat("\t", level)
					lines[i] = strings.ReplaceAll(lines[i], fromString, toString)
				}
				prevLine = line
			}
			s = strings.Join(lines, "\n")
		}

		// Should the file be saved with the executable bit enabled?
		// (Does it either start with a shebang or reside in a common bin directory like /usr/bin?)
		shebang = files.BinDirectory(filename) || strings.HasPrefix(s, "#!")

		data = []byte(s)
	}

	// Mark the data as "not changed" if it's not a binary file
	if !e.binaryFile {
		e.changed.Store(false)
	}

	// Default file mode (0644 for regular files, 0755 for executable files)
	var fileMode os.FileMode = 0o644

	// Shell scripts that contains the word "source" on the first three lines often needs to be sourced and should not be "chmod +x"-ed, nor "chmod -x" ed
	containsTheWordSource := containsInTheFirstNLines(data, 3, []byte("source "))

	// Checking the syntax highlighting makes it easy to press `ctrl-t` before saving a script,
	// to toggle the executable bit on or off. This is only for files that start with "#!".
	// Also, if the file is in one of the common bin directories, like "/usr/bin", then assume that it
	// is supposed to be executable. Also skip .install files, even though they are scripts.
	if shebang && e.syntaxHighlight && !containsTheWordSource && !strings.HasSuffix(filename, ".install") {
		// This is a script file, syntax highlighting is enabled and it does not contain the word "source "
		// (typical for shell files that should be sourced and not executed)
		fileMode = 0o755
	}

	// If it's not a binary file OR the file has changed: save the data
	if !e.binaryFile || e.changed.Load() {

		// Check if the user appears to be a quick developer
		if time.Since(editorLaunchTime) < 30*time.Second && e.mode != mode.Text && e.mode != mode.Blank {
			// Disable the quick help at start
			DisableQuickHelpScreen(nil)
		}

		// Start a spinner, in a short while
		const cursorAfterText = false
		quitChan := e.Spinner(c, tty, fmt.Sprintf("Saving %s... ", filename), fmt.Sprintf("saving %s: stopped by user", filename), 200*time.Millisecond, e.ItalicsColor, cursorAfterText)

		// Prepare gzipped data
		if strings.HasSuffix(filename, ".gz") {
			var err error
			data, err = gZipData(data)
			if err != nil {
				quitChan <- true
				return err
			}
		}

		// Save the file and return any errors
		if err := os.WriteFile(filename, data, fileMode); err != nil {
			// Stop the spinner and return
			quitChan <- true
			return err
		}

		// This file should not be considered read-only, since saving went fine
		e.readOnly = false

		// TODO: Consider the previous fileMode of the file when doing chmod +x instead of just setting 0755 or 0644

		// "chmod +x" or "chmod -x". This is needed after saving the file, in order to toggle the executable bit.
		// rust source may start with something like "#![feature(core_intrinsics)]", so avoid that.
		if !containsTheWordSource {
			if shebang && e.mode != mode.Rust && e.mode != mode.Python && e.mode != mode.Mojo && e.mode != mode.Starlark && !e.readOnly {
				// Call Chmod, but ignore errors (since this is just a bonus and not critical)
				os.Chmod(e.filename, fileMode)
				e.syntaxHighlight = true
			} else if e.mode == mode.ASCIIDoc || e.mode == mode.Just || e.mode == mode.Make || e.mode == mode.Markdown || e.mode == mode.ReStructured || e.mode == mode.SCDoc {
				fileMode = 0o644
				os.Chmod(e.filename, fileMode)
			} else if baseFilename := filepath.Base(e.filename); baseFilename == "PKGBUILD" || baseFilename == "APKBUILD" {
				fileMode = 0o644
				os.Chmod(e.filename, fileMode)
			}
		}

		// Stop the spinner
		quitChan <- true

	}

	e.redrawCursor.Store(true)

	// Trailing spaces may be trimmed, so move to the end, if needed
	if changed {
		e.GoToPosition(c, nil, *bookmark)
		if e.AfterEndOfLine() {
			e.EndNoTrim(c)
		}
		// Do the redraw manually before showing the status message
		respectOffset := true
		redrawCanvas := false
		e.HideCursorDrawLines(c, respectOffset, redrawCanvas, false)
		e.redraw.Store(false)
	}

	// All done
	return nil
}
