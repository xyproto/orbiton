package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/xyproto/files"
	"github.com/xyproto/vt100"
)

// exportScdoc tries to export the current document as a manual page, using scdoc
func (e *Editor) exportScdoc(manFilename string) error {
	scdoc := exec.Command("scdoc")

	// Place the current contents in a buffer, and feed it to stdin to the command
	var buf bytes.Buffer
	buf.WriteString(e.String())
	scdoc.Stdin = &buf

	// Create a new file and use it as stdout
	manpageFile, err := os.Create(manFilename)
	if err != nil {
		return err
	}

	var errBuf bytes.Buffer
	scdoc.Stdout = manpageFile
	scdoc.Stderr = &errBuf

	// Save the command in a temporary file
	saveCommand(scdoc)

	// Run scdoc
	if err := scdoc.Run(); err != nil {
		errorMessage := strings.TrimSpace(errBuf.String())
		if len(errorMessage) > 0 {
			return errors.New(errorMessage)
		}
		return err
	}
	return nil
}

// exportAdoc tries to export the current document as a manual page, using asciidoctor
func (e *Editor) exportAdoc(c *vt100.Canvas, tty *vt100.TTY, manFilename string) error {
	// TODO: Use a proper function for generating temporary files
	tmpfn := "___o___.adoc"
	if files.Exists(tmpfn) {
		return errors.New(tmpfn + " already exists, please remove it")
	}

	// TODO: Write a SaveAs function for the Editor
	oldFilename := e.filename
	e.filename = tmpfn
	err := e.Save(c, tty)
	if err != nil {
		e.filename = oldFilename
		return err
	}
	e.filename = oldFilename

	// Run asciidoctor
	adocCommand := exec.Command("asciidoctor", "-b", "manpage", "-o", manFilename, tmpfn)
	saveCommand(adocCommand)
	if err = adocCommand.Run(); err != nil {
		_ = os.Remove(tmpfn) // Try removing the temporary filename if pandoc fails
		return err
	}
	return os.Remove(tmpfn)
}
