package main

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/xyproto/vt100"
)

// render PDF from Markdown using pandoc
func (e *Editor) exportPandoc(c *vt100.Canvas, status *StatusBar, pandocPath, pdfFilename string) error {
	// This function used to be concurrent. There are some leftovers from this that could be refactored away.

	status.ClearAll(c)
	status.SetMessage("Exporting to PDF using Pandoc...")
	status.ShowNoTimeout(c, e)

	// The reason for writing to a temporary file is to be able to export without saving
	// the currently edited file.

	// Use the temporary directory defined in TMPDIR, with fallback to /tmp
	tempdir := os.Getenv("TMPDIR")
	if tempdir == "" {
		tempdir = "/tmp"
	}

	tempTexFilename := ""
	tf, err := ioutil.TempFile(tempdir, "__o*.tex")
	if err != nil {
		return err
	}
	tempTexFilename = tf.Name()

	tempFilename := ""
	f, err := ioutil.TempFile(tempdir, "__o*.md")
	if err != nil {
		return err
	}
	tempFilename = f.Name()

	// TODO: Write a SaveAs function for the Editor

	// Save to tmpfn
	oldFilename := e.filename
	e.filename = tempFilename
	err = e.Save(c)
	if err != nil {
		e.filename = oldFilename
		status.ClearAll(c)
		status.SetErrorMessage(err.Error())
		status.Show(c, e)
		return err
	}
	e.filename = oldFilename

	// Check if the PAPERSIZE environment variable is set
	papersize := "a4"
	if papersizeEnv := os.Getenv("PAPERSIZE"); papersizeEnv != "" {
		papersize = papersizeEnv
	}

	pandocCommand := exec.Command(pandocPath, "-fmarkdown-implicit_figures", "--toc", "-Vgeometry:left=1cm,top=1cm,right=1cm,bottom=2cm", "-Vpapersize:"+papersize, "-Vfontsize=12pt", "--pdf-engine=xelatex", "-o", pdfFilename, oldFilename)

	const listingsSetupTex = `% https://tex.stackexchange.com/a/179956/5116
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

	// TODO: Use a better temporary filename
	err = ioutil.WriteFile(tempTexFilename, []byte(listingsSetupTex), 0644)
	if err != nil {
		return err
	}

	// use the listings package
	pandocCommand.Args = append(pandocCommand.Args, "--listings", "-H"+tempTexFilename)

	// add output and input filenames
	pandocCommand.Args = append(pandocCommand.Args, "-o"+pdfFilename, oldFilename)

	// Save the command in a temporary file, using the current filename
	saveCommand(pandocCommand)

	// Use the temporary filename for the last argument, now that the command has been saved
	pandocCommand.Args[len(pandocCommand.Args)-1] = tempFilename

	if err = pandocCommand.Run(); err != nil {
		_ = os.Remove(tempFilename) // Try removing the temporary filename if pandoc fails
		status.ClearAll(c)
		status.SetErrorMessage(err.Error())
		status.Show(c, e)
		return err
	}

	// Remove the temporary file
	if err = os.Remove(tempFilename); err != nil {
		status.ClearAll(c)
		status.SetMessage(err.Error())
		status.Show(c, e)
		return err
	}

	// Remove the temporary tex file
	if err = os.Remove(tempTexFilename); err != nil {
		status.ClearAll(c)
		status.SetMessage(err.Error())
		status.Show(c, e)
		return err
	}

	status.ClearAll(c)
	status.SetMessage("Saved " + pdfFilename)
	status.ShowNoTimeout(c, e)
	return nil
}
