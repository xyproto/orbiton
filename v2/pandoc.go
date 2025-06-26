package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/vt"
)

var pandocTexFilename = filepath.Join(userConfigDir, "o", "pandoc.tex")

const (
	listingsSetupTex = `% https://tex.stackexchange.com/a/179956/5116
\usepackage{xcolor}
\lstset{
    basicstyle=\ttfamily,
    keywordstyle=\color[rgb]{0.13,0.29,0.53}\bfseries,
    stringstyle=\color[rgb]{0.31,0.60,0.02},
    commentstyle=\color[rgb]{0.56,0.35,0.01}\itshape,
    stepnumber=1,
    numbersep=5pt,
    backgroundcolor=\color[RGB]{248,248,248},
    showspaces=false,
    showstringspaces=false,
    showtabs=false,
    tabsize=2,
    captionpos=b,
    breaklines=true,
    breakatwhitespace=true,
    breakautoindent=true,
    escapeinside={\%*}{*)},
    linewidth=\textwidth,
    basewidth=0.5em,
    showlines=true,
}
`
)

// exportPandocPDF will render PDF from Markdown using pandoc
func (e *Editor) exportPandocPDF(c *vt.Canvas, tty *vt.TTY, status *StatusBar, pandocPath, pdfFilename string) error {
	// This function used to be concurrent. There are some leftovers from this that could be refactored away.

	status.ClearAll(c, true)
	status.SetMessage("Rendering to PDF using Pandoc...")
	status.ShowNoTimeout(c, e)

	// The reason for writing to a temporary file is to be able to export without saving
	// the currently edited file.

	tempFilename := ""
	f, err := os.CreateTemp(tempDir, "_o*.md")
	if err != nil {
		return err
	}
	defer os.Remove(tempFilename)
	tempFilename = f.Name()

	// TODO: Implement a SaveAs function

	// Save to tmpfn
	oldFilename := e.filename
	e.filename = tempFilename
	err = e.Save(c, tty)
	if err != nil {
		e.filename = oldFilename
		status.ClearAll(c, true)
		status.SetError(err)
		status.Show(c, e)
		return err
	}
	e.filename = oldFilename

	// Check if the PAPERSIZE environment variable is set. Default to "a4".
	papersize := env.Str("PAPERSIZE", "a4")

	pandocCommand := exec.Command(pandocPath, "-fmarkdown-implicit_figures", "--toc", "-Vgeometry:left=1cm,top=1cm,right=1cm,bottom=2cm", "-Vpapersize:"+papersize, "-Vfontsize=12pt", "--pdf-engine=xelatex", "-o", pdfFilename)

	expandedTexFilename := env.ExpandUser(pandocTexFilename)

	// Write the Pandoc Tex style file, for configuring the listings package, if it does not already exist
	if !files.Exists(expandedTexFilename) {
		// First create the folder, if needed, in a best effort attempt
		folderPath := filepath.Dir(expandedTexFilename)
		_ = os.MkdirAll(folderPath, 0o755)
		// Write the Pandoc Tex style file
		err = os.WriteFile(expandedTexFilename, []byte(listingsSetupTex), 0o644)
		if err != nil {
			status.SetErrorMessage("Could not write " + pandocTexFilename + ": " + err.Error())
			status.Show(c, e)
			return err
		}
	}

	// use the listings package
	pandocCommand.Args = append(pandocCommand.Args, "--listings", "-H"+expandedTexFilename)

	// add output and input filenames
	pandocCommand.Args = append(pandocCommand.Args, "-o"+pdfFilename, oldFilename)

	// Save the command in a temporary file, using the current filename
	saveCommand(pandocCommand)

	// Use the temporary filename for the last argument, now that the command has been saved
	pandocCommand.Args[len(pandocCommand.Args)-1] = tempFilename

	if output, err := pandocCommand.CombinedOutput(); err != nil {
		status.ClearAll(c, false)

		// The program was executed, but failed
		outputByteLines := bytes.Split(bytes.TrimSpace(output), []byte{'\n'})
		errorMessage := string(outputByteLines[len(outputByteLines)-1])

		if len(errorMessage) == 0 {
			errorMessage = err.Error()
		}

		status.SetErrorMessage(errorMessage)
		status.Show(c, e)

		return err
	}

	status.ClearAll(c, true)
	status.SetMessage("Saved " + pdfFilename)
	status.ShowNoTimeout(c, e)
	return nil
}
