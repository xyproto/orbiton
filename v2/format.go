package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xyproto/autoimport"
	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// FormatMap maps from format command to file extensions
type FormatMap map[*exec.Cmd][]string

var formatMap FormatMap

// GetFormatMap will return a map from format command to file extensions.
// It is done this way to only initialize the map once, but not at the time when the program starts.
func GetFormatMap() FormatMap {
	if formatMap == nil {
		formatMap = FormatMap{
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
			exec.Command("black"):    {".py"},
			exec.Command("rustfmt"):  {".rs"},
			exec.Command("scalafmt"): {".scala"},
			exec.Command("shfmt", "-s", "-w", "-i", "2", "-bn", "-ci", "-sr", "-kp"): {".bash", ".sh", "APKBUILD", "PKGBUILD"},
			exec.Command("v", "fmt"): {".v"},
			exec.Command("tidy", "-w", "80", "-q", "-i", "-utf8", "--show-errors", "0", "--show-warnings", "no", "--tidy-mark", "no", "-xml", "-m"): {".xml"},
			exec.Command("zig", "fmt"): {".zig"},
		}
	}
	return formatMap
}

// Using exec.Cmd instead of *exec.Cmd is on purpose, to get a new cmd.stdout and cmd.stdin every time.
func (e *Editor) formatWithUtility(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, cmd exec.Cmd, extOrBaseFilename string) error {
	if files.Which(cmd.Path) == "" { // Does the formatting tool even exist?
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
			ignoreErrors := strings.HasSuffix(cmd.Path, "tidy") && files.Which("tidy") != ""

			// Perl may place executables in /usr/bin/vendor_perl
			if e.mode == mode.Perl {
				// Use perltidy from the PATH if /usr/bin/vendor_perl/perltidy does not exists
				if cmd.Path == "/usr/bin/vendor_perl/perltidy" && !files.Exists("/usr/bin/vendor_perl/perltidy") {
					perltidyPath := files.Which("perltidy")
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

	// Format JSON
	if e.mode == mode.JSON {
		data, err := formatJSON([]byte(e.String()), jsonFormatToggle, e.indentation.PerTab)
		if err != nil {
			status.ClearAll(c)
			status.SetErrorAfterRedraw(err)
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

	// Organize Java or Kotlin imports
	if e.mode == mode.Java || e.mode == mode.Kotlin {
		const removeExistingImports = false
		const deGlobImports = true
		e.LoadBytes(organizeImports([]byte(e.String()), e.mode == mode.Java, removeExistingImports, deGlobImports))
		e.redraw = true
		// Do not return, since there is more formatting to be done
	}

	// Not in git mode, format Go or C++ code with goimports or clang-format

OUT:
	for cmd, extensions := range GetFormatMap() {
		for _, ext := range extensions {
			if strings.HasSuffix(e.filename, ext) {
				// Format a specific file instead of the current directory if "go.mod" is missing
				if sourceFilename, err := filepath.Abs(e.filename); e.mode == mode.Go && err == nil {
					sourceDir := filepath.Dir(sourceFilename)
					if !files.IsFile(filepath.Join(sourceDir, "go.mod")) {
						cmd.Args = append(cmd.Args, sourceFilename)
					}
				}
				if err := e.formatWithUtility(c, tty, status, *cmd, ext); err != nil {
					status.ClearAll(c)
					status.SetMessage(err.Error())
					status.Show(c, e)
					break OUT
				}
				break OUT
			}
		}
	}
}
