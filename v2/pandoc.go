package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/vt100"
)

// exportPandocPDF will render a PDF from Markdown using Pandoc
func (e *Editor) exportPandocPDF(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, pandocPath, pdfFilename string) error {
	status.ClearAll(c, true)
	status.SetMessage("Rendering to PDF using Pandoc...")
	status.ShowNoTimeout(c, e)

	// Create temporary Markdown file
	tempFilename := ""
	f, err := os.CreateTemp(tempDir, "_o*.md")
	if err != nil {
		return err
	}
	defer os.Remove(tempFilename)
	tempFilename = f.Name()

	// Save current buffer to temporary file
	oldFilename := e.filename
	e.filename = tempFilename
	err = e.Save(c, tty)
	if err != nil {
		e.filename = oldFilename
		status.ClearAll(c, true)
		status.SetError(err)
		status.ShowNoTimeout(c, e)
		return err
	}
	e.filename = oldFilename

	// Set paper size from environment (default: "a4")
	papersize := env.Str("PAPERSIZE", "a4")
	resourcePath := filepath.Dir(e.filename)

	// Build Pandoc command
	pandocCommand := exec.Command(
		pandocPath,
		"-f", "markdown-implicit_figures",
		"--toc",
		"-V", "geometry:left=1cm,top=1cm,right=1cm,bottom=2cm",
		"-V", "papersize:"+papersize,
		"-V", "fontsize=12pt",
		"--pdf-engine=xelatex",
		"--highlight-style=tango", // optional: improve output
		"--resource-path="+resourcePath,
		"-o", pdfFilename,
		tempFilename,
	)

	// Save the command in a temporary file, using the current filename
	saveCommand(pandocCommand)

	// Run the command and handle output
	if output, err := pandocCommand.CombinedOutput(); err != nil {
		status.ClearAll(c, false)
		outputByteLines := bytes.Split(bytes.TrimSpace(output), []byte{'\n'})
		errorMessage := string(outputByteLines[len(outputByteLines)-1])
		if len(errorMessage) == 0 {
			errorMessage = err.Error()
		}
		status.SetErrorMessage(errorMessage)
		status.ShowNoTimeout(c, e)
		return err
	}

	status.ClearAll(c, true)
	status.SetMessage("Saved " + pdfFilename)
	status.ShowNoTimeout(c, e)
	return nil
}
