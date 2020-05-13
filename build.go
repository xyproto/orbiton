package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xyproto/vt100"
)

var (
	errNoSuitableBuildCommand = errors.New("no suitable build command")
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
func (e *Editor) exportAdoc(manFilename string) error {

	// TODO: Use a proper function for generating temporary files
	tmpfn := "___o___.adoc"

	if exists(tmpfn) {
		return errors.New(tmpfn + " already exists, please remove it")
	}
	err := e.Save(&tmpfn, !e.DrawMode())
	if err != nil {
		return err
	}

	adocCommand := exec.Command("asciidoctor", "-b", "manpage", "-o", manFilename, tmpfn)

	// Run asciidoctor
	if err = adocCommand.Run(); err != nil {
		_ = os.Remove(tmpfn) // Try removing the temporary filename if pandoc fails
		return err
	}
	if err = os.Remove(tmpfn); err != nil {
		return err
	}
	return nil
}

// mustExportPandoc returns nothing, but can be used concurrently.
// This takes a bit longer than the other export types, which is why this one is different.
func (e *Editor) mustExportPandoc(c *vt100.Canvas, status *StatusBar, pandocPath, pdfFilename string) {
	status.SetMessage("Exporting to PDF using Pandoc...")
	status.ShowNoTimeout(c, e)

	// TODO: Use a proper function for generating temporary files
	tmpfn := "___o___.md"

	// Check if the temporary file already exists
	if exists(tmpfn) {
		status.ClearAll(c)
		status.SetErrorMessage(tmpfn + " already exists, please remove it")
		status.Show(c, e)
		return // from goroutine
	}

	// Create the temporary file
	err := e.Save(&tmpfn, !e.DrawMode())
	if err != nil {
		status.ClearAll(c)
		status.SetMessage(err.Error())
		status.Show(c, e)
		return // from goroutine
	}

	// TODO: Check if there are environment variables applicable to paper sizes
	pandocCommand := exec.Command(pandocPath, "-N", "--toc", "-V", "geometry:a4paper", "-o", pdfFilename, tmpfn)

	// Run pandoc
	if err = pandocCommand.Run(); err != nil {
		_ = os.Remove(tmpfn) // Try removing the temporary filename if pandoc fails
		status.ClearAll(c)
		status.SetMessage(err.Error())
		status.Show(c, e)
		return // from goroutine
	}

	// Remove the temporary file
	if err = os.Remove(tmpfn); err != nil {
		status.ClearAll(c)
		status.SetMessage(err.Error())
		status.Show(c, e)
		return // from goroutine
	}

	statusMessage := "Saved " + pdfFilename
	status.ClearAll(c)
	status.SetMessage(statusMessage)
	status.Show(c, e)
}

// BuildOrExport will try to build the source code or export the document.
// Returns a status message and then true if an action was performed and another true if compilation worked out.
func (e *Editor) BuildOrExport(c *vt100.Canvas, status *StatusBar, filename string) (string, bool, bool) {
	ext := filepath.Ext(filename)

	// scdoc
	manFilename := "out.1"
	if ext == ".scd" || ext == ".scdoc" {
		if err := e.exportScdoc(manFilename); err != nil {
			return err.Error(), true, false
		}
		return "Exported " + manFilename, true, true
	}

	// asciidoctor
	if ext == ".adoc" {
		if err := e.exportAdoc(manFilename); err != nil {
			return err.Error(), true, false
		}
		return "Exported " + manFilename, true, true
	}

	// pandoc
	if pandocPath := which("pandoc"); ext == ".md" && e.mode == modeMarkdown && pandocPath != "" {
		pdfFilename := strings.Replace(filepath.Base(filename), ".", "_", -1) + ".pdf"
		// Export to PDF using pandoc, concurrently.
		go e.mustExportPandoc(c, status, pandocPath, pdfFilename)
		// TODO: Add a minimum of error detection. Perhaps wait just 20ms and check if the goroutine is still running.
		return "", true, true // no message returned, the mustExportPandoc function handles it's own status output
	}

	defaultExecutableName := "main" // If the current directory name is not found

	// Find a suitable default executable name (only used with OCaml)
	if ext == ".ml" {
		if curdir, err := os.Getwd(); err == nil { // no error
			defaultExecutableName = filepath.Base(curdir)
		}
	}

	// Set up a few variables
	var (
		// Map from build command to a list of file extensions (or basenames for files without an extension)
		build = map[*exec.Cmd][]string{
			exec.Command("go", "build"):                                     {".go"},
			exec.Command("cxx"):                                             {".cpp", ".cc", ".cxx", ".h", ".hpp", ".c++", ".h++", ".c"},
			exec.Command("zig", "build"):                                    {".zig"},
			exec.Command("v", filename):                                     {".v"},
			exec.Command("cargo", "build"):                                  {".rs"},
			exec.Command("ghc", "-dynamic", filename):                       {".hs"},
			exec.Command("makepkg"):                                         {"PKGBUILD"},
			exec.Command("python", "-m", "py_compile", filename):            {".py"}, // Compile to .pyc
			exec.Command("ocamlopt", "-o", defaultExecutableName, filename): {".ml"}, // OCaml
		}
	)

	// Check if one of the build commands are applicable for this filename
	baseFilename := filepath.Base(filename)
	var foundCommand *exec.Cmd
	for command, exts := range build {
		for _, ext := range exts {
			if strings.HasSuffix(filename, ext) || baseFilename == ext {
				foundCommand = command
			}
		}
	}

	// Can not export nor compile, nothing more to do
	if foundCommand == nil {
		return errNoSuitableBuildCommand.Error(), false, false
	}

	// --- Compilation ---

	var (
		cmd                   = foundCommand // shorthand
		progressStatusMessage = "Building"
		testingInstead        bool
	)

	// Special per-language considerations
	if ext == ".rs" && (!exists("Cargo.toml") && !exists("../Cargo.toml")) {
		// Use rustc instead of cargo if Cargo.toml is missing and the extension is .rs
		cmd = exec.Command("rustc", filename)
	} else if (ext == ".cc" || ext == ".h") && exists("BUILD.bazel") {
		// Google-style C++ + Bazel projects
		cmd = exec.Command("bazel", "build")
	} else if ext == ".zig" && !exists("build.zig") {
		// Just build the current file
		cmd = exec.Command("zig", "build-exe", "-lc", filename)
	} else if strings.HasSuffix(filename, "_test.go") {
		// If it's a test-file, run the test instead of building
		cmd = exec.Command("go", "test")
		progressStatusMessage = "Testing"
		testingInstead = true
	}

	// Display a status message with no timeout, about what is currently being done
	if status != nil {
		status.ClearAll(c)
		status.SetMessage(progressStatusMessage)
		status.ShowNoTimeout(c, e)
	}

	// Run the command and fetch the combined output from stderr and stdout.
	// Ignore the status code / error, only look at the output.
	output, err := cmd.CombinedOutput()

	// NOTE: Don't do anything with the output and err variables here, let the if below handle it.

	// Did the command return a non-zero status code, or does the output contain "error:"?
	if err != nil || bytes.Contains(output, []byte("error:")) { // failed tests also end up here

		// This is not for Go, since the word "error:" may not appear when there are errors

		errorMessage := "Build error"

		errorMarker := "error:"
		if testingInstead {
			errorMarker = "FAIL:"
		}

		if e.mode == modePython {
			if errorLine, errorMessage := ParsePythonError(string(output), filepath.Base(filename)); errorLine != -1 {
				e.redraw = e.GoTo(LineIndex(errorLine-1), c, status)
				return "Error: " + errorMessage, true, false
			}
		}

		// Find the first error message
		lines := strings.Split(string(output), "\n")

		var prevLine string
		for _, line := range lines {
			if ext == ".hs" {
				if strings.Contains(prevLine, errorMarker) {
					if errorMessage = strings.TrimSpace(line); strings.HasPrefix(errorMessage, "• ") {
						errorMessage = string([]rune(errorMessage)[2:])
						break
					}
				}
			} else if strings.Contains(line, errorMarker) {
				parts := strings.SplitN(line, errorMarker, 2)
				errorMessage = strings.TrimSpace(parts[1])
				break
			}
			prevLine = line
		}

		if testingInstead {
			errorMessage = "Test failed: " + errorMessage
		}

		// NOTE: Don't return here even if errorMessage contains an error message

		// Analyze all lines
		for i, line := range lines {
			// Go, C++ and Haskell
			if strings.Count(line, ":") >= 3 {
				fields := strings.SplitN(line, ":", 4)
				// Go to Y:X, if available
				var foundY LineIndex
				if y, err := strconv.Atoi(fields[1]); err == nil { // no error
					foundY = LineIndex(y - 1)
					e.redraw = e.GoTo(foundY, c, status)
					e.redrawCursor = e.redraw
					foundX := -1
					if x, err := strconv.Atoi(fields[2]); err == nil { // no error
						foundX = x - 1
					}
					if foundX != -1 {
						tabs := strings.Count(e.Line(foundY), "\t")
						e.pos.sx = foundX + (tabs * (e.spacesPerTab - 1))
						e.Center(c)
						// Use the error message as the status message
						if len(fields) >= 4 {
							if ext != ".hs" {
								return strings.Join(fields[3:], " "), true, false
							}
							// TODO: Test this code path
							return "UNIMPLEMENTED 1", true, false
						}
					}
				}
				return "UNIMPLEMENTED 2", true, false
			} else if (i-1) > 0 && (i-1) < len(lines) {
				// Rust
				if msgLine := lines[i-1]; strings.Contains(line, " --> ") && strings.Count(line, ":") == 2 && strings.Count(msgLine, ":") >= 1 {
					errorFields := strings.SplitN(msgLine, ":", 2)                  // Already checked for 2 colons
					errorMessage := strings.TrimSpace(errorFields[1])               // There will always be 3 elements in errorFields, so [1] is fine
					locationFields := strings.SplitN(line, ":", 3)                  // Already checked for 2 colons in line
					filenameFields := strings.SplitN(locationFields[0], " --> ", 2) // [0] is fine, already checked for " ---> "
					errorFilename := strings.TrimSpace(filenameFields[1])           // [1] is fine
					if filename != errorFilename {
						return "Error in " + errorFilename + ": " + errorMessage, true, false
					}
					errorY := locationFields[1]
					errorX := locationFields[2]

					// Go to Y:X, if available
					var foundY LineIndex
					if y, err := strconv.Atoi(errorY); err == nil { // no error
						foundY = LineIndex(y - 1)
						e.redraw = e.GoTo(foundY, c, status)
						e.redrawCursor = e.redraw
						foundX := -1
						if x, err := strconv.Atoi(errorX); err == nil { // no error
							foundX = x - 1
						}
						if foundX != -1 {
							tabs := strings.Count(e.Line(foundY), "\t")
							e.pos.sx = foundX + (tabs * (e.spacesPerTab - 1))
							e.Center(c)
							// Use the error message as the status message
							if errorMessage != "" {
								return errorMessage, true, false
							}
						}
					}
					e.redrawCursor = true
					break
				}
			}
		}
	} else {
		// The command failed (status code != 0), but no errors were picked up from the output
		return "Could not compile", true, false
	}

	// No status message, no error, not a success?
	return "", false, false
}
