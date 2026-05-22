package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/mode"
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

// exportPandocPDF will render PDF from Markdown or reStructuredText using pandoc.
// The caller should set a "Rendering..." messageAfterRedraw first.
// Success and error results are reported via SetMessageAfterRedraw.
func (e *Editor) exportPandocPDF(c *vt.Canvas, tty *vt.TTY, status *StatusBar, pandocPath, pdfFilename string) error {
	// Write to a temporary file so unsaved changes are rendered

	// Choose the temp file extension and pandoc input format based on mode
	tempPattern := "_o*.md"
	inputFormat := "markdown-implicit_figures"
	if e.mode == mode.ReStructured {
		tempPattern = "_o*.rst"
		inputFormat = "rst"
	}

	f, err := os.CreateTemp(tempDir, tempPattern)
	if err != nil {
		status.SetErrorMessageAfterRedraw("pandoc tempfile: " + err.Error())
		e.redraw.Store(true)
		return err
	}
	f.Close()
	tempFilename := f.Name()
	defer os.Remove(tempFilename)

	// TODO: Implement a SaveAs function

	// Save to tmpfn
	oldFilename := e.filename
	e.filename = tempFilename
	err = e.Save(c, tty)
	e.filename = oldFilename
	if err != nil {
		status.SetErrorMessageAfterRedraw("pandoc save temp: " + err.Error())
		e.redraw.Store(true)
		return err
	}

	// Check if the PAPERSIZE environment variable is set. Default to "a4".
	papersize := env.Str("PAPERSIZE", "a4")

	expandedTexFilename := env.ExpandUser(pandocTexFilename)

	// Write the Pandoc Tex style file, for configuring the listings package, if it does not already exist
	if !files.Exists(expandedTexFilename) {
		// First create the folder, if needed, in a best effort attempt
		folderPath := filepath.Dir(expandedTexFilename)
		_ = os.MkdirAll(folderPath, 0o755)
		// Write the Pandoc Tex style file
		err = os.WriteFile(expandedTexFilename, []byte(listingsSetupTex), 0o644)
		if err != nil {
			status.SetErrorMessageAfterRedraw("Could not write " + pandocTexFilename + ": " + err.Error())
			return err
		}
	}

	// Build the pandoc argument list once, so saveCommand and the
	// executed command stay in sync
	pandocArgs := []string{
		"-f" + inputFormat,
		"--toc",
		"-Vgeometry:left=1cm,top=1cm,right=1cm,bottom=2cm",
		"-Vpapersize:" + papersize,
		"-Vfontsize=12pt",
		"--pdf-engine=xelatex",
	}

	// The listings package is incompatible with RST roles (:role:`text`)
	// which pandoc translates to \lstinline[role=X]!text! — an invalid key.
	if e.mode != mode.ReStructured {
		pandocArgs = append(pandocArgs, "--listings", "-H"+expandedTexFilename)
	}

	pandocArgs = append(pandocArgs, "-o", pdfFilename)

	// Save the runnable command with the real filename, for last_command.sh
	savedArgs := append(append([]string(nil), pandocArgs...), oldFilename)
	saveCommand(exec.Command(pandocPath, savedArgs...))

	// Run pandoc against the temporary file, so unsaved changes are rendered
	pandocCommand := exec.Command(pandocPath, append(pandocArgs, tempFilename)...)

	if output, err := pandocCommand.CombinedOutput(); err != nil {
		// Pandoc ran but failed. Report the last line of stderr.
		outputByteLines := bytes.Split(bytes.TrimSpace(output), []byte{'\n'})
		errorMessage := ""
		if len(outputByteLines) > 0 {
			errorMessage = string(outputByteLines[len(outputByteLines)-1])
		}
		if len(errorMessage) == 0 {
			errorMessage = err.Error()
		}
		status.SetErrorMessageAfterRedraw("pandoc: " + errorMessage)
		e.redraw.Store(true)
		return err
	}

	status.SetMessageAfterRedraw("Wrote " + pdfFilename)
	e.redraw.Store(true)
	return nil
}
