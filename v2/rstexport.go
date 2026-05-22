package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
	"github.com/xyproto/vt"
)

// exportRSTHTML will render HTML from reStructuredText using rst2html or pandoc
func (e *Editor) exportRSTHTML(c *vt.Canvas, status *StatusBar, htmlFilename string) error {
	status.ClearAll(c, true)
	status.SetMessage("Rendering RST to HTML...")
	status.ShowNoTimeout(c, e)

	// Get the current content of the editor
	rstContent := e.String()

	// Write to a temporary file
	f, err := os.CreateTemp(tempDir, "_o*.rst")
	if err != nil {
		status.ClearAll(c, false)
		status.SetError(err)
		status.Show(c, e)
		return err
	}
	f.Close()
	tempFilename := f.Name()
	defer os.Remove(tempFilename)

	if err := os.WriteFile(tempFilename, []byte(rstContent), 0o644); err != nil {
		status.ClearAll(c, false)
		status.SetError(err)
		status.Show(c, e)
		return err
	}

	// Try rst2html5 first, then rst2html, then pandoc
	var cmd *exec.Cmd
	if rstTool := files.WhichCached("rst2html5"); rstTool != "" {
		cmd = exec.Command(rstTool, tempFilename, htmlFilename)
	} else if rstTool := files.WhichCached("rst2html"); rstTool != "" {
		cmd = exec.Command(rstTool, tempFilename, htmlFilename)
	} else if pandocPath := files.WhichCached("pandoc"); pandocPath != "" {
		cmd = exec.Command(pandocPath, "-frst", "-thtml", "-s", "-o", htmlFilename, tempFilename)
	} else {
		status.ClearAll(c, false)
		status.SetErrorMessage("Please install docutils or pandoc for RST export")
		status.Show(c, e)
		return nil
	}

	saveCommand(cmd)

	if output, err := cmd.CombinedOutput(); err != nil {
		outputLines := strings.Split(strings.TrimSpace(string(output)), "\n")
		errorMessage := ""
		if len(outputLines) > 0 {
			errorMessage = outputLines[len(outputLines)-1]
		}
		if errorMessage == "" {
			errorMessage = err.Error()
		}
		status.ClearAll(c, false)
		status.SetErrorMessage("rst2html: " + errorMessage)
		status.Show(c, e)
		return err
	}

	status.ClearAll(c, true)
	status.SetMessage("Saved " + filepath.Base(htmlFilename))
	status.ShowNoTimeout(c, e)
	return nil
}
