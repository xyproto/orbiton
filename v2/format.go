package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/klauspost/asmfmt"
	"github.com/xyproto/autoimport"
	"github.com/xyproto/files"
	"github.com/xyproto/lookslikegoasm"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// FormatMap maps from format command to file extensions
type FormatMap map[mode.Mode]*exec.Cmd

var formatMap FormatMap

// InstallMissingTools will try to install some of the tools, if they are missing
func (e *Editor) InstallMissingTools() {
	switch e.mode {
	case mode.Go:
		if files.Which("go") != "" && files.Which("goimport") == "" {
			run("go install golang.org/x/tools/cmd/goimports@latest")
		}
	}
}

// GetFormatMap will return a map from format command to file extensions.
// It is done this way to only initialize the map once, but not at the time when the program starts.
func (e *Editor) GetFormatMap() FormatMap {
	if formatMap == nil {
		formatMap = FormatMap{
			mode.Cpp:        exec.Command("clang-format", "-fallback-style=WebKit", "-style=file", "-i", "--"),
			mode.C:          exec.Command("clang-format", "-fallback-style=WebKit", "-style=file", "-i", "--"),
			mode.ObjC:       exec.Command("clang-format", "-fallback-style=WebKit", "-style=file", "-i", "--"),
			mode.CS:         exec.Command("astyle", "--mode=cs"),
			mode.Crystal:    exec.Command("crystal", "tool", "format"),
			mode.CSS:        exec.Command("prettier", "--tab-width", "2", "-w"),
			mode.Dart:       exec.Command("dart", "format"),
			mode.Go:         exec.Command("goimports", "-w", "--"),
			mode.Haskell:    exec.Command("brittany", "--write-mode=inplace"),
			mode.Java:       exec.Command("google-java-format", "-a", "-i"),
			mode.JavaScript: exec.Command("prettier", "--tab-width", "4", "-w"),
			mode.TypeScript: exec.Command("prettier", "--tab-width", "4", "-w"),
			mode.Just:       exec.Command("just", "--unstable", "--fmt", "-f"),
			mode.Kotlin:     exec.Command("ktlint", "-F"),
			mode.Lua:        exec.Command("lua-format", "-i", "--no-keep-simple-function-one-line", "--column-limit=120", "--indent-width=2", "--no-use-tab"),
			mode.OCaml:      exec.Command("ocamlformat"),
			mode.Odin:       exec.Command("odinfmt", "-w"),
			mode.Perl:       exec.Command("/usr/bin/vendor_perl/perltidy", "-se", "-b", "-i=2", "-ole=unix", "-bt=2", "-pt=2", "-sbt=2", "-ce"),
			mode.Python:     exec.Command("black"),
			mode.Rust:       exec.Command("rustfmt"),
			mode.Scala:      exec.Command("scalafmt"),
			mode.Shell:      exec.Command("shfmt", "-s", "-w", "-i", "2", "-bn", "-ci", "-sr", "-kp"),
			mode.V:          exec.Command("v", "fmt"),
			mode.XML:        exec.Command("tidy", "-w", "80", "-q", "-i", "-utf8", "--show-errors", "0", "--show-warnings", "no", "--tidy-mark", "no", "-xml", "-m"),
			mode.Zig:        exec.Command("zig", "fmt"),
			mode.PHP:        exec.Command("php-cs-fixer", "--no-ansi", "--no-interaction", "fix"),
		}
	}
	return formatMap
}

// Using exec.Cmd instead of *exec.Cmd is on purpose, to get a new cmd.stdout and cmd.stdin every time.
func (e *Editor) formatWithUtility(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, cmd exec.Cmd) error {
	if files.WhichCached(cmd.Path) == "" { // Does the formatting tool even exist?
		return errors.New(cmd.Path + " is missing")
	}

	tempFirstName := "o"
	if e.mode == mode.Kotlin {
		tempFirstName = "O"
	}

	extOrBaseFilename := filepath.Ext(e.filename)
	if extOrBaseFilename == "" {
		extOrBaseFilename = filepath.Base(e.filename)
	}

	if f, err := os.CreateTemp(tempDir, tempFirstName+".*"+extOrBaseFilename); err == nil {
		// no error, everything is fine
		tempFilename := f.Name()
		defer os.Remove(tempFilename)
		defer f.Close()

		// TODO: Implement e.SaveAs
		oldFilename := e.filename
		e.filename = tempFilename
		err := e.Save(c, tty)
		e.filename = oldFilename

		if err == nil {
			// Add the filename of the temporary file to the command
			cmd.Args = append(cmd.Args, tempFilename)

			// Save the command in a temporary file
			saveCommand(&cmd)

			// Format the temporary file
			output, err := cmd.CombinedOutput()

			// Ignore errors if the command is "tidy" and tidy exists
			ignoreErrors := strings.HasSuffix(cmd.Path, "tidy") && files.WhichCached("tidy") != ""

			// Perl may place executables in /usr/bin/vendor_perl
			if e.mode == mode.Perl {
				// Use perltidy from the PATH if /usr/bin/vendor_perl/perltidy does not exists
				if cmd.Path == "/usr/bin/vendor_perl/perltidy" && !files.Exists("/usr/bin/vendor_perl/perltidy") {
					perltidyPath := files.WhichCached("perltidy")
					if perltidyPath == "" {
						return errors.New("perltidy is missing")
					}
					cmd.Path = perltidyPath
				}
			}

			if err != nil && !ignoreErrors {
				// Only grab the first error message
				errorMessage := strings.TrimSpace(string(output))
				if errorMessage == "" && err != nil {
					errorMessage = err.Error()
				}
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
						redraw, _ := e.GoTo(LineIndex(foundY), c, status)
						e.redraw.Store(redraw)
						foundX := -1
						if x, err := strconv.Atoi(fields[2]); err == nil { // no error
							foundX = x - 1
						}
						if foundX != -1 {
							tabs := strings.Count(e.Line(LineIndex(foundY)), "\t")
							e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
							e.Center(c)
						}
					}
					e.redrawCursor.Store(true)
				}
				return retErr
			}

			if _, err := e.Load(c, tty, FilenameOrData{tempFilename, []byte{}, 0, false}); err != nil {
				return err
			}
			// Mark the data as changed, despite just having loaded a file
			e.changed.Store(true)
			e.redrawCursor.Store(true)
		}
		// Try to close the file. f.Close() checks if f is nil before closing.
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
	}
	return nil
}

// oneLine returns true if the given bytes appear to be only one line of text, or less
func oneLine(data []byte) bool {
	return bytes.Count(data, []byte{'\n'}) <= 1
}

// formatJSON can format the given JSON data
func formatJSON(data []byte, jsonFormatToggle *bool, indentationPerTab int) ([]byte, error) {
	var v interface{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}
	// Format the JSON bytes, first without indentation and then
	// with indentation.
	var indentedJSON []byte
	if *jsonFormatToggle {
		indentedJSON, err = json.Marshal(v)
		*jsonFormatToggle = false
	} else {
		indentationString := strings.Repeat(" ", indentationPerTab)
		indentedJSON, err = json.MarshalIndent(v, "", indentationString)
		*jsonFormatToggle = true
	}
	if err != nil {
		return nil, err
	}

	// This is a hack to prevent the json.Unmarshal formatter to end up formatting everything on one line
	if oneLine(indentedJSON) && !oneLine(data) { // did everything end up on a single line
		// Try again (TODO: Figure out why this is sometimes needed)
		indentedJSON, err = formatJSON(indentedJSON, jsonFormatToggle, indentationPerTab)
		if err != nil {
			return nil, err
		}
		// Is it still on a single line?
		if oneLine(indentedJSON) {
			// Ignore the formatting changes and just return the original data
			return data, nil
		}
	}

	return indentedJSON, nil
}

// organizeImports can fix, sort and organize imports for Kotlin and for Java
func organizeImports(data []byte, onlyJava, removeExistingImports, deGlob bool) []byte {
	ima, err := autoimport.New(onlyJava, removeExistingImports, deGlob)
	if err != nil {
		return data // no change
	}
	const verbose = false
	newData, err := ima.FixImports(data, verbose)
	if err != nil {
		return data // no change
	}
	return newData
}

func (e *Editor) formatCode(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, jsonFormatToggle *bool) {
	switch e.mode {
	case mode.JSON: // Format JSON
		data, err := formatJSON([]byte(e.String()), jsonFormatToggle, e.indentation.PerTab)
		if err != nil {
			status.ClearAll(c, true)
			status.SetErrorAfterRedraw(err)
			return
		}
		e.LoadBytes(data)
		e.redraw.Store(true)
		return
	case mode.FSTAB: // Format /etc/fstab files
		const spaces = 2
		e.LoadBytes(formatFstab([]byte(e.String()), spaces))
		e.redraw.Store(true)
		return
	case mode.Assembly:
		if !lookslikegoasm.Consider(e.String()) {
			break // no formatter for regular Assembly, yet
		}
		e.mode = mode.GoAssembly
		fallthrough
	case mode.GoAssembly:
		if formatted, err := asmfmt.Format(strings.NewReader(e.String())); err == nil { // success
			e.LoadBytes(formatted)
			e.redraw.Store(true)
			return // All done
		}
	case mode.Java, mode.Kotlin:
		const removeExistingImports = false
		const deGlobImports = true
		e.LoadBytes(organizeImports([]byte(e.String()), e.mode == mode.Java, removeExistingImports, deGlobImports))
		e.redraw.Store(true)
		// Do not return, since there is more formatting to be done
	}

	e.InstallMissingTools()

	// Not in git mode, format Go or C++ code with goimports or clang-format
	for formatMode, cmd := range e.GetFormatMap() {
		if e.mode == formatMode && e.mode == mode.Go {
			// Format a specific file instead of the current directory if "go.mod" is missing
			if sourceFilename, err := filepath.Abs(e.filename); err == nil {
				sourceDir := filepath.Dir(sourceFilename)
				if !files.IsFile(filepath.Join(sourceDir, "go.mod")) {
					cmd.Args = append(cmd.Args, sourceFilename)
				}
			}
		}
		if e.mode == formatMode {
			if err := e.formatWithUtility(c, tty, status, *cmd); err != nil {
				status.ClearAll(c, false)
				status.SetMessage(err.Error())
				status.Show(c, e)
				break
			}
			break
		}
	}
}
