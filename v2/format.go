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

	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
	"github.com/yosssi/gohtml"
)

// Map from formatting command to a list of file extensions
var format = map[*exec.Cmd][]string{
	exec.Command("clang-format", "-fallback-style=WebKit", "-style=file", "-i", "--"): {".c", ".c++", ".cc", ".cpp", ".cxx", ".h", ".h++", ".hpp"},
	exec.Command("astyle", "--mode=cs"):                                               {".cs"},
	exec.Command("crystal", "tool", "format"):                                         {".cr"},
	exec.Command("prettier", "--tab-width", "2", "-w"):                                {".css"},
	exec.Command("dart", "format"):                                                    {".dart"},
	exec.Command("goimports", "-w", "--"):                                             {".go"},
	exec.Command("brittany", "--write-mode=inplace"):                                  {".hs"},
	exec.Command("google-java-format", "-a", "-i"):                                    {".java"},
	exec.Command("prettier", "--tab-width", "4", "-w"):                                {".js", ".ts"},
	exec.Command("just", "--unstable", "--fmt", "-f"):                                 {".just", ".justfile", "justfile"},
	exec.Command("ktlint", "-F"):                                                      {".kt", ".kts"},
	exec.Command("lua-format", "-i", "--no-keep-simple-function-one-line", "--column-limit=120", "--indent-width=2", "--no-use-tab"): {".lua"},
	exec.Command("ocamlformat"): {".ml"},
	exec.Command("/usr/bin/vendor_perl/perltidy", "-se", "-b", "-i=2", "-ole=unix", "-bt=2", "-pt=2", "-sbt=2", "-ce"): {".pl"},
	exec.Command("autopep8", "-i", "--max-line-length", "120"):                                                         {".py"},
	exec.Command("rustfmt"):  {".rs"},
	exec.Command("scalafmt"): {".scala"},
	exec.Command("shfmt", "-s", "-w", "-i", "2", "-bn", "-ci", "-sr", "-kp"): {".bash", ".sh", "APKBUILD", "PKGBUILD"},
	exec.Command("v", "fmt"): {".v"},
	exec.Command("tidy", "-w", "80", "-q", "-i", "-utf8", "--show-errors", "0", "--show-warnings", "no", "--tidy-mark", "no", "-xml", "-m"): {".xml"},
	exec.Command("zig", "fmt"): {".zig"},
}

// Using exec.Cmd instead of *exec.Cmd is on purpose, to get a new cmd.stdout and cmd.stdin every time.
func (e *Editor) formatWithUtility(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, cmd exec.Cmd, extOrBaseFilename string) error {
	if which(cmd.Path) == "" { // Does the formatting tool even exist?
		return errors.New(cmd.Path + " is missing")
	}

	tempFirstName := "o"
	if e.mode == mode.Kotlin {
		tempFirstName = "O"
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
							e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
							e.Center(c)
						}
					}
					e.redrawCursor = true
				}
				return retErr
			}

			if _, err := e.Load(c, tty, FilenameOrData{tempFilename, []byte{}, 0, false}); err != nil {
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

// formatFstab can format the contents of /etc/fstab files. The suggested number of spaces is 2.
func formatFstab(data []byte, spaces int) []byte {
	var (
		buf       bytes.Buffer
		nl        = []byte{'\n'}
		longest   = make(map[int]int) // The longest length of a field, for each field index
		byteLines = bytes.Split(data, nl)
	)

	// Find the longest field length for each field on each line
	for _, line := range byteLines {
		trimmedLine := bytes.TrimSpace(line)
		if len(trimmedLine) == 0 || bytes.HasPrefix(trimmedLine, []byte{'#'}) {
			continue
		}
		// Find the longest field length for each field
		for i, field := range bytes.Fields(trimmedLine) {
			fieldLength := len(string(field))
			if val, ok := longest[i]; ok {
				if fieldLength > val {
					longest[i] = fieldLength
				}
			} else {
				longest[i] = fieldLength
			}
		}
	}

	// Format the lines nicely
	for _, line := range byteLines {
		trimmedLine := bytes.TrimSpace(line)
		if len(trimmedLine) == 0 {
			continue
		}
		if bytes.HasPrefix(trimmedLine, []byte{'#'}) { // Output comments as they are, but trimmed
			buf.Write(trimmedLine)
			buf.Write(nl)
		} else { // Format the fields
			for i, field := range bytes.Fields(trimmedLine) {
				fieldLength := len(string(field))
				padCount := spaces // Space between the fields if all fields have equal length
				if longest[i] > fieldLength {
					padCount += longest[i] - fieldLength
				}
				buf.Write(field)
				if padCount > 0 {
					buf.Write(bytes.Repeat([]byte{' '}, padCount))
				}
			}
			buf.Write(nl)
		}
	}
	return buf.Bytes()
}

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
	return indentedJSON, nil
}

func formatHTML(data []byte) ([]byte, error) {
	return gohtml.FormatBytes(data), nil
}

func (e *Editor) formatCode(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, jsonFormatToggle *bool) {

	// Format JSON
	if e.mode == mode.JSON {
		data, err := formatJSON([]byte(e.String()), jsonFormatToggle, e.indentation.PerTab)
		if err != nil {
			status.ClearAll(c)
			status.ShowErrorAfterRedraw(err)
			return
		}
		e.LoadBytes(data)
		e.redraw = true
		return
	}

	// Format HTML
	if e.mode == mode.HTML {
		data, err := formatHTML([]byte(e.String()))
		if err != nil {
			status.ClearAll(c)
			status.ShowErrorAfterRedraw(err)
			return
		}
		e.LoadBytes(data)
		e.redraw = true
		return
	}

	// Format /etc/fstab files
	if baseFilename := filepath.Base(e.filename); baseFilename == "fstab" {
		const spaces = 2
		e.LoadBytes(formatFstab([]byte(e.String()), spaces))
		e.redraw = true
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
