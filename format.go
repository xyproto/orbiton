package main

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xyproto/env"
	"github.com/xyproto/vt100"
)

func (e *Editor) formatWithUtility(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, cmd *exec.Cmd, extOrBaseFilename string) error {
	if which(cmd.Path) == "" { // Does the formatting tool even exist?
		return errors.New(cmd.Path + " is missing")
	}

	utilityName := filepath.Base(cmd.Path)
	status.SetMessage("Calling " + utilityName)
	status.Show(c, e)

	// Use the temporary directory defined in TMPDIR, with fallback to /tmp
	tempdir := env.Str("TMPDIR", "/tmp")

	if f, err := ioutil.TempFile(tempdir, "__o*"+extOrBaseFilename); err == nil {
		// no error, everything is fine
		tempFilename := f.Name()

		// TODO: Implement e.SaveAs
		oldFilename := e.filename
		e.filename = tempFilename
		err := e.Save(c)
		e.filename = oldFilename

		if err == nil {
			// Add the filename of the temporary file to the command
			cmd.Args = append(cmd.Args, tempFilename)

			// Save the command in a temporary file
			saveCommand(cmd)

			// Format the temporary file
			output, err := cmd.CombinedOutput()

			// Ignore errors if the command is "tidy" and tidy exists
			ignoreErrors := strings.HasSuffix(cmd.Path, "tidy") && which("tidy") != ""

			if err != nil && !ignoreErrors {
				// Only grab the first error message
				errorMessage := strings.TrimSpace(string(output))
				if strings.Count(errorMessage, "\n") > 0 {
					errorMessage = strings.TrimSpace(strings.SplitN(errorMessage, "\n", 2)[0])
				}
				var retErr error
				if errorMessage == "" {
					retErr = errors.New("failed to format code")
				} else {
					retErr = errors.New("failed to format code: " + errorMessage)
				}
				if strings.Count(errorMessage, ":") >= 3 {
					fields := strings.Split(errorMessage, ":")
					// Go To Y:X, if available
					var foundY int
					if y, err := strconv.Atoi(fields[1]); err == nil { // no error
						foundY = y - 1
						e.redraw = e.GoTo(LineIndex(foundY), c, status)
						foundX := -1
						if x, err := strconv.Atoi(fields[2]); err == nil { // no error
							foundX = x - 1
						}
						if foundX != -1 {
							tabs := strings.Count(e.Line(LineIndex(foundY)), "\t")
							e.pos.sx = foundX + (tabs * (e.tabs.spacesPerTab - 1))
							e.Center(c)
						}
					}
					e.redrawCursor = true
				}
				return retErr
			}

			if _, err := e.Load(c, tty, tempFilename); err != nil {
				return err
			}
			// Mark the data as changed, despite just having loaded a file
			e.changed = true
			e.redrawCursor = true

			// Try to remove the temporary file regardless if "goimports -w" worked out or not
			_ = os.Remove(tempFilename)
		}
		// Try to close the file. f.Close() checks if f is nil before closing.
		_ = f.Close()
		e.redraw = true
		e.redrawCursor = true
	}
	return nil
}
