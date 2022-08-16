package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xyproto/env"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// Map from formatting command to a list of file extensions
var format = map[*exec.Cmd][]string{
	exec.Command("goimports", "-w", "--"):                                             {".go"},
	exec.Command("clang-format", "-fallback-style=WebKit", "-style=file", "-i", "--"): {".cpp", ".cc", ".cxx", ".h", ".hpp", ".c++", ".h++", ".c"},
	exec.Command("zig", "fmt"):                                                        {".zig"},
	exec.Command("v", "fmt"):                                                          {".v"},
	exec.Command("rustfmt"):                                                           {".rs"},
	exec.Command("brittany", "--write-mode=inplace"):                                  {".hs"},
	exec.Command("autopep8", "-i", "--max-line-length", "120"):                        {".py"},
	exec.Command("ocamlformat"):                                                       {".ml"},
	exec.Command("crystal", "tool", "format"):                                         {".cr"},
	exec.Command("ktlint", "-F"):                                                      {".kt", ".kts"},
	exec.Command("google-java-format", "-i"):                                          {".java"},
	exec.Command("scalafmt"):                                                          {".scala"},
	exec.Command("astyle", "--mode=cs"):                                               {".cs"},
	exec.Command("prettier", "--tab-width", "4", "-w"):                                {".js", ".ts"},
	exec.Command("prettier", "--tab-width", "2", "-w"):                                {".css"},
	exec.Command("shfmt", "-s", "-w", "-i", "2", "-bn", "-ci", "-sr", "-kp"):          {".sh", ".bash", "PKGBUILD"},
	exec.Command("lua-format", "-i", "--no-keep-simple-function-one-line", "--column-limit=120", "--indent-width=2", "--no-use-tab"):                                                                                                 {".lua"},
	exec.Command("tidy", "-w", "80", "-q", "-i", "-utf8", "--show-errors", "0", "--show-warnings", "no", "--tidy-mark", "no", "-xml", "-m"):                                                                                          {".xml"},
	exec.Command("tidy", "-w", "120", "-q", "-i", "-utf8", "--show-errors", "0", "--show-warnings", "no", "--tidy-mark", "no", "--hide-endtags", "yes", "--force-output", "yes", "-ashtml", "-omit", "no", "-xml", "no", "-m", "-c"): {".html", ".htm"},
	exec.Command("/usr/bin/vendor_perl/perltidy", "-se", "-b", "-i=2", "-ole=unix", "-bt=2", "-pt=2", "-sbt=2", "-ce"):                                                                                                               {".pl"},
}

// Using exec.Cmd instead of *exec.Cmd is on purpose, to get a new cmd.stdout and cmd.stdin every time.
func (e *Editor) formatWithUtility(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, cmd exec.Cmd, extOrBaseFilename string) error {
	if which(cmd.Path) == "" { // Does the formatting tool even exist?
		return errors.New(cmd.Path + " is missing")
	}

	//utilityName := filepath.Base(cmd.Path)
	//status.SetMessage("Calling " + utilityName)
	//status.Show(c, e)

	// Use the temporary directory defined in TMPDIR, with fallback to /tmp
	tempdir := env.Dir("TMPDIR", "/tmp")

	if f, err := os.CreateTemp(tempdir, "o.*"+extOrBaseFilename); err == nil {
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
			ignoreErrors := strings.HasSuffix(cmd.Path, "tidy") && which("tidy") != ""

			// Perl may place executables in /usr/bin/vendor_perl
			if e.mode == mode.Perl {
				// Use perltidy from the PATH if /usr/bin/vendor_perl/perltidy does not exists
				if cmd.Path == "/usr/bin/vendor_perl/perltidy" && !exists("/usr/bin/vendor_perl/perltidy") {
					perltidyPath := which("perltidy")
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
						e.redraw, _ = e.GoTo(LineIndex(foundY), c, status)
						foundX := -1
						if x, err := strconv.Atoi(fields[2]); err == nil { // no error
							foundX = x - 1
						}
						if foundX != -1 {
							tabs := strings.Count(e.Line(LineIndex(foundY)), "\t")
							e.pos.sx = foundX + (tabs * (e.tabsSpaces.PerTab - 1))
							e.Center(c)
						}
					}
					e.redrawCursor = true
				}
				return retErr
			}

			if _, err := e.Load(c, tty, FilenameOrData{tempFilename, []byte{}, 0}); err != nil {
				return err
			}
			// Mark the data as changed, despite just having loaded a file
			e.changed = true
			e.redrawCursor = true
		}
		// Try to close the file. f.Close() checks if f is nil before closing.
		e.redraw = true
		e.redrawCursor = true
	}
	return nil
}

func (e *Editor) formatCode(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, jsonFormatToggle *bool) {

	// Format JSON
	if e.mode == mode.JSON {
		var v any

		err := json.Unmarshal([]byte(e.String()), &v)
		if err != nil {
			status.ClearAll(c)
			status.ShowErrorAfterRedraw(err)
			return
		}

		// Format the JSON bytes, first without indentation and then
		// with indentation.
		var indentedJSON []byte
		if *jsonFormatToggle {
			indentedJSON, err = json.Marshal(v)
			*jsonFormatToggle = false
		} else {
			indentationString := strings.Repeat(" ", e.tabsSpaces.PerTab)
			indentedJSON, err = json.MarshalIndent(v, "", indentationString)
			*jsonFormatToggle = true
		}
		if err != nil {
			status.ClearAll(c)
			status.ShowErrorAfterRedraw(err)
			return
		}

		e.LoadBytes(indentedJSON)
		e.redraw = true
		return
	}

	baseFilename := filepath.Base(e.filename)
	if baseFilename == "fstab" {
		cmd := exec.Command("fstabfmt", "-i")
		if which(cmd.Path) == "" { // Does the formatting tool even exist?
			status.ClearAll(c)
			status.SetErrorMessage(cmd.Path + " is missing")
			status.Show(c, e)
			return
		}
		if err := e.formatWithUtility(c, tty, status, *cmd, baseFilename); err != nil {
			status.ClearAll(c)
			status.SetMessage(err.Error())
			status.Show(c, e)
		}
		return
	}

	// Not in git mode, format Go or C++ code with goimports or clang-format

OUT:
	for cmd, extensions := range format {
		for _, ext := range extensions {
			if strings.HasSuffix(e.filename, ext) {
				if err := e.formatWithUtility(c, tty, status, *cmd, ext); err != nil {
					status.ClearAll(c)
					status.SetMessage(err.Error())
					status.Show(c, e)
				}
				break OUT
			}
		}
	}

}
