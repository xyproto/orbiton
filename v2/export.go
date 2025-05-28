package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/xyproto/files"
	"github.com/xyproto/vt100"
)

// exportScdoc tries to export the current document as a manual page, using scdoc
func (e *Editor) exportScdoc(manFilename string) error {
	scdocPath := files.WhichCached("scdoc")
	if scdocPath == "" {
		return errors.New("could not find scdoc in the PATH")
	}

	scdoc := exec.Command(scdocPath)

	// Place the current contents in a buffer, and feed it by stdin to the command
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
	err = scdoc.Run()
	if err != nil {
		errorMessage := strings.TrimSpace(errBuf.String())
		if len(errorMessage) > 0 {
			err = errors.New(errorMessage)
		}
	}
	return err
}

// exportAdoc tries to export the current document as a manual page, using asciidoctor
func (e *Editor) exportAdoc(c *vt100.Canvas, tty *vt100.TTY, manFilename string) error {
	adocPath := files.WhichCached("asciidoctor")
	if adocPath == "" {
		return errors.New("could not find asciidoctor in the PATH")
	}
	tmpfile, err := os.CreateTemp("", "*.adoc")
	if err != nil {
		return err
	}
	tmpfn := tmpfile.Name()

	defer func() {
		tmpfile.Close()
		os.Remove(tmpfn)
	}()

	if _, err := io.WriteString(tmpfile, e.String()); err != nil {
		return err
	}

	// Run asciidoctor
	adocCommand := exec.Command(adocPath, "-b", "manpage", "-o", manFilename, tmpfn)
	saveCommand(adocCommand)
	if err := adocCommand.Run(); err != nil {
		_ = os.Remove(tmpfn) // Try removing the temporary filename if asciidoctor fails
		return err
	}
	return os.Remove(tmpfn)
}
