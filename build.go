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
func (e *Editor) exportAdoc(c *vt100.Canvas, manFilename string) error {

	// TODO: Use a proper function for generating temporary files
	tmpfn := "___o___.adoc"
	if exists(tmpfn) {
		return errors.New(tmpfn + " already exists, please remove it")
	}

	// TODO: Write a SaveAs function for the Editor
	oldFilename := e.filename
	e.filename = tmpfn
	err := e.Save(c)
	if err != nil {
		e.filename = oldFilename
		return err
	}
	e.filename = oldFilename

	// Run asciidoctor
	adocCommand := exec.Command("asciidoctor", "-b", "manpage", "-o", manFilename, tmpfn)
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
	status.ClearAll(c)
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

	// TODO: Write a SaveAs function for the Editor

	// Save to tmpfn
	oldFilename := e.filename
	e.filename = tmpfn
	err := e.Save(c)
	if err != nil {
		e.filename = oldFilename
		status.ClearAll(c)
		status.SetErrorMessage(err.Error())
		status.Show(c, e)
		return // from goroutine
	}
	e.filename = oldFilename

	// TODO: Check if there are environment variables applicable to paper sizes
	// TODO: -N is maybe not needed ?
	pandocCommand := exec.Command(pandocPath, "-N", "-fmarkdown-implicit_figures", "--toc", "-V", "\"geometry:a4paper\"", "-V", "\"geometry:margin=.4in\"", "-o", pdfFilename, tmpfn)
	// Run pandoc
	//panic(pandocCommand)
	if err = pandocCommand.Run(); err != nil {
		_ = os.Remove(tmpfn) // Try removing the temporary filename if pandoc fails
		status.ClearAll(c)
		status.SetErrorMessage(err.Error())
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

	status.ClearAll(c)
	status.SetMessage("Saved " + pdfFilename)
	status.ShowNoTimeout(c, e)
}

// BuildOrExport will try to build the source code or export the document.
// Returns a status message and then true if an action was performed and another true if compilation/testing worked out.
func (e *Editor) BuildOrExport(c *vt100.Canvas, status *StatusBar, filename string) (string, bool, bool) {
	if status != nil {
		status.Clear(c)
	}

	ext := filepath.Ext(filename)

	// scdoc
	manFilename := "out.1"
	if ext == ".scd" || ext == ".scdoc" {
		if err := e.exportScdoc(manFilename); err != nil {
			return err.Error(), true, false
		}
		return "Saved " + manFilename, true, true
	}

	// asciidoctor
	if ext == ".adoc" {
		if err := e.exportAdoc(c, manFilename); err != nil {
			return err.Error(), true, false
		}
		return "Saved " + manFilename, true, true
	}

	// pandoc
	if pandocPath := which("pandoc"); e.mode == modeMarkdown && pandocPath != "" {
		pdfFilename := strings.Replace(filepath.Base(filename), ".", "_", -1) + ".pdf"
		// Export to PDF using pandoc, concurrently. The goroutine handles its own status messages.
		go e.mustExportPandoc(c, status, pandocPath, pdfFilename)
		// TODO: Add a minimum of error detection. Perhaps wait just 20ms and check if the goroutine is still running.
		return "", true, true // no message returned, the mustExportPandoc function handles it's own status output
	}

	exeFirstName := "main" // If the current directory name is not found

	// Find a suitable default executable first name
	if e.mode == modeOCaml || e.mode == modeKotlin || e.mode == modeLua {
		if curdir, err := os.Getwd(); err == nil { // no error
			exeFirstName = filepath.Base(curdir)
		}
	}

	javaShellCommand := "javaFiles=$(find . -type f -name '*.java'); for f in $javaFiles; do grep -q 'static void main' \"$f\" && mainJavaFile=\"$f\"; done; className=$(grep -oP '(?<=class )[A-Z]+[a-z,A-Z,0-9]*' \"$mainJavaFile\" | head -1); packageName=$(grep -oP '(?<=package )[a-z,A-Z,0-9,.]*' \"$mainJavaFile\" | head -1); if [[ $packageName != \"\" ]]; then packageName=\"$packageName.\"; fi; mkdir -p _o_build/META-INF; javac -d _o_build $javaFiles; cd _o_build; echo \"Main-Class: $packageName$className\" > META-INF/MANIFEST.MF; classFiles=$(find . -type f -name '*.class'); jar cmf META-INF/MANIFEST.MF ../main.jar $classFiles; cd ..; rm -rf _o_build"

	// TODO: Change the map to not use file extensions, but rather rely on the modes from ftdetect.go

	// Set up a few variables
	var (
		// Map from build command to a list of file extensions (or basenames for files without an extension)
		build = map[*exec.Cmd][]string{
			exec.Command("go", "build"):                                                      {".go"},                                                     // Go
			exec.Command("cxx"):                                                              {".cpp", ".cc", ".cxx", ".h", ".hpp", ".c++", ".h++", ".c"}, // C++ and C
			exec.Command("zig", "build"):                                                     {".zig"},                                                    // Zig
			exec.Command("v", filename):                                                      {".v"},                                                      // V
			exec.Command("cargo", "build"):                                                   {".rs"},                                                     // Rust
			exec.Command("ghc", "-dynamic", filename):                                        {".hs"},                                                     // Haskell
			exec.Command("python", "-m", "py_compile", filename):                             {".py"},                                                     // Python, compile to .pyc
			exec.Command("ocamlopt", "-o", exeFirstName, filename):                           {".ml"},                                                     // OCaml
			exec.Command("crystal", "build", "--no-color", filename):                         {".cr"},                                                     // Crystal
			exec.Command("kotlinc", filename, "-include-runtime", "-d", exeFirstName+".jar"): {".kt"},                                                     // Kotlin, build a .jar file
			exec.Command("sh", "-c", javaShellCommand):                                       {".java"},                                                   // Java, build a .jar file
			exec.Command("luac", "-o", exeFirstName+".out", filename):                        {".lua"},                                                    // Lua, build an .out file
			exec.Command("nim", "c", filename):                                               {".nim"},                                                    // Nim
			exec.Command("fpc", filename):                                                    {".pp", ".pas", ".lpr"},                                     // Object Pascal / Delphi
		}
	)

	// Check if one of the build commands are applicable for this filename
	baseFilename := filepath.Base(filename)
	var foundCommand *exec.Cmd
	for command, exts := range build {
		for _, ext := range exts {
			if strings.HasSuffix(filename, ext) || baseFilename == ext {
				foundCommand = command
				// TODO: also check that the executable in the command exists
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
		kotlinNative          bool
	)

	// Special per-language considerations
	if e.mode == modeRust && (!exists("Cargo.toml") && !exists("../Cargo.toml")) {
		// Use rustc instead of cargo if Cargo.toml is missing and the extension is .rs
		if which("rustc") != "" {
			cmd = exec.Command("rustc", filename)
		}
	} else if (ext == ".cc" || ext == ".h") && exists("BUILD.bazel") {
		// Google-style C++ + Bazel projects
		if which("bazel") != "" {
			cmd = exec.Command("bazel", "build")
		}
	} else if e.mode == modeZig && !exists("build.zig") {
		// Just build the current file
		if which("zig") != "" {
			cmd = exec.Command("zig", "build-exe", "-lc", filename)
		}
	} else if strings.HasSuffix(filename, "_test.go") {
		// If it's a test-file, run the test instead of building
		if which("go") != "" {
			cmd = exec.Command("go", "test", "-failfast")
		}
		progressStatusMessage = "Testing"
		testingInstead = true
	} else if e.mode == modeKotlin && which("kotlinc-native") != "" {
		kotlinNative = true
		cmd = exec.Command("kotlinc-native", "-nowarn", "-opt", "-Xallocator=mimalloc", "-produce", "program", "-linker-option", "--as-needed", filename, "-o", exeFirstName)
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
	//panic(cmd.String() + "OUTPUT: " + string(output))

	outputString := string(bytes.TrimSpace(output))

	if err != nil && len(outputString) == 0 {
		errorMessage := "Error: no output"
		// TODO: Also add checks for other executables
		switch {
		case e.mode == modeZig && which("zig") == "":
			errorMessage = "Error: the Zig compiler is not installed"
		}
		// Could not run, and there was no output. Perhaps the executable is missing?
		return errorMessage, true, false
	}

	if kotlinNative && exists(exeFirstName+".kexe") {
		//panic("rename " + exeFirstName + ".kexe" + " -> " + exeFirstName)
		os.Rename(exeFirstName+".kexe", exeFirstName)
	}

	// NOTE: Don't do anything with the output and err variables here, let the if below handle it.

	errorMarker := "error:"
	if testingInstead {
		errorMarker = ": "
	} else if e.mode == modeCrystal || e.mode == modeObjectPascal {
		errorMarker = "Error:"
	}

	if e.mode == modeZig && bytes.Contains(output, []byte("nrecognized glibc version")) {
		byteLines := bytes.Split(output, []byte("\n"))
		fields := strings.Split(string(byteLines[0]), ":")
		errorMessage := "Error: unrecognized glibc version"
		if len(fields) > 1 {
			errorMessage += ": " + strings.TrimSpace(fields[1])
		}
		return errorMessage, true, false
	}

	if e.mode == modeGo {
		switch {
		case bytes.Contains(output, []byte(": undefined")):
			errorMarker = "undefined"
		case bytes.Contains(output, []byte(": error")):
			errorMarker = "error"
		case bytes.Count(output, []byte(":")) >= 2:
			errorMarker = ":"
		}
	}

	// Did the command return a non-zero status code, or does the output contain "error:"?
	if err != nil || bytes.Contains(output, []byte(errorMarker)) { // failed tests also end up here

		// This is not for Go, since the word "error:" may not appear when there are errors

		errorMessage := "Build error"
		if testingInstead {
			errorMessage = "Test error"
		}

		if e.mode == modePython {
			if errorLine, errorMessage := ParsePythonError(string(output), filepath.Base(filename)); errorLine != -1 {
				e.redraw = e.GoTo(LineIndex(errorLine-1), c, status)
				return "Error: " + errorMessage, true, false
			}
		}

		// Find the first error message
		var (
			lines               = strings.Split(string(output), "\n")
			prevLine            string
			crystalLocationLine string
		)
		for _, line := range lines {
			if e.mode == modeHaskell {
				if strings.Contains(prevLine, errorMarker) {
					if errorMessage = strings.TrimSpace(line); strings.HasPrefix(errorMessage, "• ") {
						errorMessage = string([]rune(errorMessage)[2:])
						break
					}
				}
			} else if e.mode == modeCrystal {
				if strings.HasPrefix(line, "Error:") {
					errorMessage = line[6:]
					if len(crystalLocationLine) > 0 {
						break
					}
				} else if strings.HasPrefix(line, "In ") {
					crystalLocationLine = line
				}
			} else if e.mode == modeObjectPascal {
				errorMessage = ""
				if strings.Contains(line, " Error: ") {
					pos := strings.Index(line, " Error: ")
					errorMessage = line[pos+8:]
				} else if strings.Contains(line, " Fatal: ") {
					pos := strings.Index(line, " Fatal: ")
					errorMessage = line[pos+8:]
				}
				if len(errorMessage) > 0 {
					parts := strings.SplitN(line, "(", 2)
					errorFilename, rest := parts[0], parts[1]
					baseErrorFilename := filepath.Base(errorFilename)
					parts = strings.SplitN(rest, ",", 2)
					lineNumberString, rest := parts[0], parts[1]
					parts = strings.SplitN(rest, ")", 2)
					lineColumnString, rest := parts[0], parts[1]
					errorMessage = rest

					// Move to (x, y), line number first and then column number
					if i, err := strconv.Atoi(lineNumberString); err == nil {
						foundY := LineIndex(i - 1)
						e.redraw = e.GoTo(foundY, c, status)
						e.redrawCursor = e.redraw
						if x, err := strconv.Atoi(lineColumnString); err == nil { // no error
							foundX := x - 1
							tabs := strings.Count(e.Line(foundY), "\t")
							e.pos.sx = foundX + (tabs * (e.tabs.spacesPerTab - 1))
							e.Center(c)
						}
					}

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "In " + baseErrorFilename + ": " + errorMessage, true, false
					}
					return errorMessage, true, false
				}
			} else if e.mode == modeLua {
				if strings.Contains(line, " error near ") && strings.Count(line, ":") >= 3 {
					parts := strings.SplitN(line, ":", 4)
					errorMessage = parts[3]

					if i, err := strconv.Atoi(parts[2]); err == nil {
						foundY := LineIndex(i - 1)
						e.redraw = e.GoTo(foundY, c, status)
						e.redrawCursor = e.redraw
					}

					baseErrorFilename := filepath.Base(parts[1])
					if baseErrorFilename != baseFilename {
						return "In " + baseErrorFilename + ": " + errorMessage, true, false
					}
					return errorMessage, true, false
				}
				break
			} else if e.mode == modeGo && errorMarker == ":" && strings.Count(line, ":") >= 2 {
				parts := strings.SplitN(line, ":", 2)
				errorMessage = strings.Join(parts[2:], ":")
				break
			} else if strings.Contains(line, errorMarker) {
				parts := strings.SplitN(line, errorMarker, 2)
				if errorMarker == "undefined" {
					errorMessage = errorMarker + strings.TrimSpace(parts[1])
				} else {
					errorMessage = strings.TrimSpace(parts[1])
				}
				if testingInstead {
					e.redrawCursor = true
				}
				break
			}
			prevLine = line
		}

		if testingInstead {
			errorMessage = "Test failed: " + errorMessage
			return errorMessage, true, false
		}

		if e.mode == modeCrystal {
			// Crystal has the location on a different line from the error message
			fields := strings.Split(crystalLocationLine, ":")
			if len(fields) != 3 {
				return errorMessage, true, false
			}
			if y, err := strconv.Atoi(fields[1]); err == nil { // no error

				foundY := LineIndex(y - 1)
				e.redraw = e.GoTo(foundY, c, status)
				e.redrawCursor = e.redraw

				if x, err := strconv.Atoi(fields[2]); err == nil { // no error
					foundX := x - 1
					tabs := strings.Count(e.Line(foundY), "\t")
					e.pos.sx = foundX + (tabs * (e.tabs.spacesPerTab - 1))
					e.Center(c)
				}

			}
			return errorMessage, true, false
		}

		// NOTE: Don't return here even if errorMessage contains an error message

		// Analyze all lines
		for i, line := range lines {
			// Go, C++, Haskell, Kotlin and more
			if strings.Count(line, ":") >= 3 && (strings.Contains(line, "error:") || strings.Contains(line, errorMarker)) {
				fields := strings.SplitN(line, ":", 4)
				baseErrorFilename := filepath.Base(fields[0])
				// Check if the filenames are matching, or if the error is in a different file
				if baseErrorFilename != baseFilename {
					return "In " + baseErrorFilename + ": " + strings.TrimSpace(fields[3]), true, false
				}
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
						e.pos.sx = foundX + (tabs * (e.tabs.spacesPerTab - 1))
						e.Center(c)

						// Use the error message as the status message
						if len(fields) >= 4 {
							if ext != ".hs" {
								return strings.Join(fields[3:], " "), true, false
							}
							return errorMessage, true, false
						}
					}
				}
				return errorMessage, true, false
			} else if (i > 0) && i < (len(lines)-1) {
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
							e.pos.sx = foundX + (tabs * (e.tabs.spacesPerTab - 1))
							e.Center(c)
							// Use the error message as the status message
							if errorMessage != "" {
								return errorMessage, true, false
							}
						}
					}
					e.redrawCursor = true
					// Nope, just the error message
					//return errorMessage, true, false
				}
			}
		}
	}
	if e.mode == modePython {
		return "Syntax OK", true, true
	}

	// Could not interpret the error message, return the last line of the output
	if err != nil && len(outputString) > 0 {
		outputLines := strings.Split(outputString, "\n")
		lastLine := outputLines[len(outputLines)-1]
		return "Error: " + lastLine, false, false
	}

	return "Success", true, true
}
