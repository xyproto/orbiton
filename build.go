package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

var (
	errNoSuitableBuildCommand = errors.New("no suitable build command")
	zigCacheDir               = filepath.Join(userCacheDir, "o/zig")
	pandocMutex               sync.RWMutex
)

// BuildOrExport will try to build the source code or export the document.
// Returns a status message and then true if an action was performed and another true if compilation/testing worked out.
// Will also return the executable output file, if available after compilation.
func (e *Editor) BuildOrExport(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, filename string, background bool) (string, bool, bool, string) {
	if status != nil {
		status.Clear(c)
	}

	ext := filepath.Ext(filename)

	// scdoc
	manFilename := "out.1"
	if ext == ".scd" || ext == ".scdoc" {
		if err := e.exportScdoc(manFilename); err != nil {
			return err.Error(), true, false, ""
		}
		return "Saved " + manFilename, true, true, manFilename
	}

	// asciidoctor
	if ext == ".adoc" {
		if err := e.exportAdoc(c, tty, manFilename); err != nil {
			return err.Error(), true, false, ""
		}
		return "Saved " + manFilename, true, true, manFilename
	}

	// pandoc
	if pandocPath := which("pandoc"); e.mode == mode.Markdown && pandocPath != "" {
		pdfFilename := strings.Replace(filepath.Base(filename), ".", "_", -1) + ".pdf"
		// Export to PDF using pandoc. The function handles its own status messages.
		// TODO: Don't ignore the error
		if background {
			go func() {
				pandocMutex.Lock()
				_ = e.exportPandoc(c, tty, status, pandocPath, pdfFilename)
				pandocMutex.Unlock()
			}()
		} else {
			_ = e.exportPandoc(c, tty, status, pandocPath, pdfFilename)
		}
		// TODO: Add a minimum of error detection. Perhaps wait just 20ms and check if the goroutine is still running.
		return "", true, true, "" // no message returned, the mustExportPandoc function handles it's own status output
	}

	exeFirstName := "main" // If the current directory name is not found

	// Find a suitable default executable first name
	if e.mode == mode.OCaml || e.mode == mode.Kotlin || e.mode == mode.Lua {
		if curdir, err := os.Getwd(); err == nil { // no error
			exeFirstName = filepath.Base(curdir)
		}
	}

	javaShellCommand := "javaFiles=$(find . -type f -name '*.java'); for f in $javaFiles; do grep -q 'static void main' \"$f\" && mainJavaFile=\"$f\"; done; className=$(grep -oP '(?<=class )[A-Z]+[a-z,A-Z,0-9]*' \"$mainJavaFile\" | head -1); packageName=$(grep -oP '(?<=package )[a-z,A-Z,0-9,.]*' \"$mainJavaFile\" | head -1); if [[ $packageName != \"\" ]]; then packageName=\"$packageName.\"; fi; mkdir -p _o_build/META-INF; javac -d _o_build $javaFiles; cd _o_build; echo \"Main-Class: $packageName$className\" > META-INF/MANIFEST.MF; classFiles=$(find . -type f -name '*.class'); jar cmf META-INF/MANIFEST.MF ../main.jar $classFiles; cd ..; rm -rf _o_build"

	scalaShellCommand := "scalaFiles=$(find . -type f -name '*.scala'); for f in $scalaFiles; do grep -q 'def main' \"$f\" && mainScalaFile=\"$f\"; grep -q ' extends App ' \"$f\" && mainScalaFile=\"$f\"; done; objectName=$(grep -oP '(?<=object )[A-Z]+[a-z,A-Z,0-9]*' \"$mainScalaFile\" | head -1); packageName=$(grep -oP '(?<=package )[a-z,A-Z,0-9,.]*' \"$mainScalaFile\" | head -1); if [[ $packageName != \"\" ]]; then packageName=\"$packageName.\"; fi; mkdir -p _o_build/META-INF; scalac -d _o_build $scalaFiles; cd _o_build; echo -e \"Main-Class: $packageName$objectName\\nClass-Path: /usr/share/scala/lib/scala-library.jar\" > META-INF/MANIFEST.MF; classFiles=$(find . -type f -name '*.class'); jar cmf META-INF/MANIFEST.MF ../main.jar $classFiles; cd ..; rm -rf _o_build"

	// Compile directly to jar with scalac if /usr/share/scala/lib/scala-library.jar is not found
	if !exists("/usr/share/scala/lib/scala-library.jar") {
		scalaShellCommand = "scalac -d run_with_scala.jar $(find . -type f -name '*.scala')"
	}

	// For building a .jar file that can not be run with "java -jar main.jar" but with "scala main.jar": scalac -jar main.jar Hello.scala

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
			exec.Command("lein", "uberjar"):                                                  {".clj", ".cljs", ".clojure"},                               // Clojure
			exec.Command("ghc", "-dynamic", filename):                                        {".hs"},                                                     // Haskell
			exec.Command("python", "-m", "py_compile", filename):                             {".py"},                                                     // Python, compile to .pyc
			exec.Command("ocamlopt", "-o", exeFirstName, filename):                           {".ml"},                                                     // OCaml
			exec.Command("crystal", "build", "--no-color", filename):                         {".cr"},                                                     // Crystal
			exec.Command("kotlinc", filename, "-include-runtime", "-d", exeFirstName+".jar"): {".kt", ".kts"},                                             // Kotlin, build a .jar file
			exec.Command("sh", "-c", scalaShellCommand):                                      {".scala"},                                                  // Scala, build a .jar file
			exec.Command("sh", "-c", javaShellCommand):                                       {".java"},                                                   // Java, build a .jar file
			exec.Command("luac", "-o", exeFirstName+".out", filename):                        {".lua"},                                                    // Lua, build an .out file
			exec.Command("nim", "c", filename):                                               {".nim"},                                                    // Nim
			exec.Command("fpc", filename):                                                    {".pp", ".pas", ".lpr"},                                     // Object Pascal / Delphi
			exec.Command("gdc", "-o", exeFirstName, filename):                                {".d"},                                                      // D
			exec.Command("xdg-open", filename):                                               {".htm", ".html"},                                           // Display HTML in the browser
			exec.Command("odin", "build", filename):                                          {".odin"},                                                   // Odin
			exec.Command("csc", "-nologo", "-unsafe", filename):                              {".cs"},                                                     // C#
			exec.Command("mlton", filename):                                                  {".sml"},
		}
	)

	// Check if one of the build commands are applicable for this filename
	baseFilename := filepath.Base(filename)
	var foundCommand exec.Cmd // exec.Cmd instead of *exec.Cmd, on purpose, to get a new stdin and stdout every time
	found := false
	for command, exts := range build {
		for _, ext := range exts {
			if strings.HasSuffix(filename, ext) || baseFilename == ext {
				foundCommand = *command
				found = true
				// TODO: also check that the executable in the command exists
			}
		}
	}

	// Can not export nor compile, nothing more to do
	if !found {
		return errNoSuitableBuildCommand.Error(), false, false, ""
	}

	// --- Compilation ---

	var (
		cmd                   = foundCommand // shorthand
		progressStatusMessage = "Building"
		kotlinNative          bool
	)

	if e.mode == mode.HTML || e.mode == mode.XML {
		progressStatusMessage = "Displaying"
	}

	baseDirName := exeFirstName
	absFilename, err := filepath.Abs(filename)
	if err == nil { // success
		dirName := filepath.Dir(absFilename)
		baseDirName = filepath.Base(dirName)
	}

	// Special per-language considerations
	if e.mode == mode.Rust && (!exists("Cargo.toml") && !exists("../Cargo.toml")) {
		// Use rustc instead of cargo if Cargo.toml is missing and the extension is .rs
		if which("rustc") != "" {

			cmd = *exec.Command("rustc", filename, "-o", baseDirName)
		}
	} else if e.mode == mode.Clojure && !exists("project.clj") && exists("../project.clj") {
		cmd.Path = filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))
	} else if e.mode == mode.Clojure && !exists("project.clj") && !exists("../project.clj") && exists("../project.clj") {
		cmd.Path = filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	} else if (ext == ".cc" || ext == ".h") && exists("BUILD.bazel") {
		// Google-style C++ + Bazel projects
		if which("bazel") != "" {
			cmd = *exec.Command("bazel", "build")
		}
	} else if e.mode == mode.Zig && !exists("build.zig") {
		// Just build the current file
		if which("zig") != "" {
			sourceCode := ""
			sourceData, err := ioutil.ReadFile(filename)
			if err == nil { // success
				sourceCode = string(sourceData)
			}
			cmd = *exec.Command("zig", "build-exe", "-lc", filename, "--name", baseDirName, "--cache-dir", zigCacheDir)
			// TODO: Find a better way than this
			if strings.Contains(sourceCode, "SDL2/SDL.h") {
				cmd.Args = append(cmd.Args, "-lSDL2")
			}
			if strings.Contains(sourceCode, "gmp.h") {
				cmd.Args = append(cmd.Args, "-lgmp")
			}
			if strings.Contains(sourceCode, "glfw") {
				cmd.Args = append(cmd.Args, "-lglfw")
			}
		}
	} else if e.mode == mode.Kotlin && which("kotlinc-native") != "" {
		kotlinNative = true
		cmd = *exec.Command("kotlinc-native", "-nowarn", "-opt", "-Xallocator=mimalloc", "-produce", "program", "-linker-option", "--as-needed", filename, "-o", exeFirstName)
	}

	// Display a status message with no timeout, about what is currently being done
	if status != nil {
		status.ClearAll(c)
		status.SetMessage(progressStatusMessage)
		status.ShowNoTimeout(c, e)
	}

	// Save the command in a temporary file
	saveCommand(&cmd)

	// Run the command and fetch the combined output from stderr and stdout.
	// Ignore the status code / error, only look at the output.
	output, err := cmd.CombinedOutput()
	//panic(cmd.String() + "OUTPUT: " + string(output))

	outputString := string(bytes.TrimSpace(output))

	if err != nil && len(outputString) == 0 {
		errorMessage := "Error: no output"
		// TODO: Also add checks for other executables
		switch {
		case e.mode == mode.Zig && which("zig") == "":
			errorMessage = "Error: the Zig compiler is not installed"
		}
		// Could not run, and there was no output. Perhaps the executable is missing?
		return errorMessage, true, false, ""
	}

	if kotlinNative && exists(exeFirstName+".kexe") {
		//panic("rename " + exeFirstName + ".kexe" + " -> " + exeFirstName)
		os.Rename(exeFirstName+".kexe", exeFirstName)
	}

	// NOTE: Don't do anything with the output and err variables here, let the if below handle it.

	errorMarker := "error:"
	if e.mode == mode.Crystal || e.mode == mode.ObjectPascal || e.mode == mode.StandardML {
		errorMarker = "Error:"
	} else if e.mode == mode.CS {
		errorMarker = ": error "
	}

	if e.mode == mode.Zig && bytes.Contains(output, []byte("nrecognized glibc version")) {
		byteLines := bytes.Split(output, []byte("\n"))
		fields := strings.Split(string(byteLines[0]), ":")
		errorMessage := "Error: unrecognized glibc version"
		if len(fields) > 1 {
			errorMessage += ": " + strings.TrimSpace(fields[1])
		}
		return errorMessage, true, false, ""
	}

	if e.mode == mode.Go {
		switch {
		case bytes.Contains(output, []byte(": undefined")):
			errorMarker = "undefined"
		case bytes.Contains(output, []byte(": error")):
			errorMarker = "error"
		case bytes.Contains(output, []byte("go: cannot find main module")):
			errorMessage := "no main module, try go mod init"
			return errorMessage, true, false, ""
		case bytes.Contains(output, []byte("go: ")):
			byteLines := bytes.SplitN(output[4:], []byte("\n"), 2)
			errorMessage := "error: " + string(byteLines[0])
			return errorMessage, true, false, ""
		case bytes.Count(output, []byte(":")) >= 2:
			errorMarker = ":"
		}
	} else if e.mode == mode.Odin {
		switch {
		case bytes.Contains(output, []byte(") ")):
			errorMarker = ") "
		}
	} else if err == nil && (e.mode == mode.HTML || e.mode == mode.XML) {
		return "Success", true, true, ""
	}

	// Did the command return a non-zero status code, or does the output contain "error:"?
	if err != nil || bytes.Contains(output, []byte(errorMarker)) { // failed tests also end up here

		// This is not for Go, since the word "error:" may not appear when there are errors

		errorMessage := "Build error"

		if e.mode == mode.Python {
			if errorLine, errorMessage := ParsePythonError(string(output), filepath.Base(filename)); errorLine != -1 {
				e.redraw = e.GoTo(LineIndex(errorLine-1), c, status)
				return "Error: " + errorMessage, true, false, ""
			}
		}

		// Find the first error message
		var (
			lines               = strings.Split(string(output), "\n")
			prevLine            string
			crystalLocationLine string
		)
		for _, line := range lines {
			if e.mode == mode.Haskell {
				if strings.Contains(prevLine, errorMarker) {
					if errorMessage = strings.TrimSpace(line); strings.HasPrefix(errorMessage, "• ") {
						errorMessage = string([]rune(errorMessage)[2:])
						break
					}
				}
			} else if e.mode == mode.StandardML {
				if strings.Contains(prevLine, errorMarker) && strings.Contains(prevLine, ".") {
					errorMessage = strings.TrimSpace(line)
					fields := strings.Split(prevLine, " ")
					if len(fields) > 2 {
						location := fields[2]
						fields = strings.Split(location, "-")
						if len(fields) > 0 {
							location = fields[0]
							locCol := strings.Split(location, ".")
							if len(locCol) > 0 {
								lineNumberString := locCol[0]
								lineColumnString := locCol[1]
								// Move to (x, y), line number first and then column number
								if i, err := strconv.Atoi(lineNumberString); err == nil {
									foundY := LineIndex(i - 1)
									e.redraw = e.GoTo(foundY, c, status)
									e.redrawCursor = e.redraw
									if x, err := strconv.Atoi(lineColumnString); err == nil { // no error
										foundX := x - 1
										tabs := strings.Count(e.Line(foundY), "\t")
										e.pos.sx = foundX + (tabs * (e.tabsSpaces.PerTab - 1))
										e.Center(c)
									}
								}
								return errorMessage, true, false, ""
							}
						}
					}
					break
				}
			} else if e.mode == mode.Crystal {
				if strings.HasPrefix(line, "Error:") {
					errorMessage = line[6:]
					if len(crystalLocationLine) > 0 {
						break
					}
				} else if strings.HasPrefix(line, "In ") {
					crystalLocationLine = line
				}
			} else if e.mode == mode.Odin {
				errorMessage = ""
				if strings.Contains(line, errorMarker) {
					whereAndWhat := strings.SplitN(line, errorMarker, 2)
					where := whereAndWhat[0]
					errorMessage = whereAndWhat[1]
					filenameAndLoc := strings.SplitN(where, "(", 2)
					errorFilename := filenameAndLoc[0]
					baseErrorFilename := filepath.Base(errorFilename)
					loc := filenameAndLoc[1]
					locCol := strings.SplitN(loc, ":", 2)
					lineNumberString := locCol[0]
					lineColumnString := locCol[1]

					// Move to (x, y), line number first and then column number
					if i, err := strconv.Atoi(lineNumberString); err == nil {
						foundY := LineIndex(i)
						e.redraw = e.GoTo(foundY, c, status)
						e.redrawCursor = e.redraw
						if x, err := strconv.Atoi(lineColumnString); err == nil { // no error
							foundX := x - 1
							tabs := strings.Count(e.Line(foundY), "\t")
							e.pos.sx = foundX + (tabs * (e.tabsSpaces.PerTab - 1))
							e.Center(c)
						}
					}

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "In " + baseErrorFilename + ": " + errorMessage, true, false, ""
					}
					return errorMessage, true, false, ""
				}
			} else if e.mode == mode.ObjectPascal || e.mode == mode.CS {
				errorMessage = ""
				if strings.Contains(line, " Error: ") {
					pos := strings.Index(line, " Error: ")
					errorMessage = line[pos+8:]
				} else if strings.Contains(line, " Fatal: ") {
					pos := strings.Index(line, " Fatal: ")
					errorMessage = line[pos+8:]
				} else if strings.Contains(line, ": error ") {
					pos := strings.Index(line, ": error ")
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
					if e.mode == mode.CS {
						if strings.Count(rest, ":") == 2 {
							parts := strings.SplitN(rest, ":", 3)
							errorMessage = parts[2]
						}
					}

					// Move to (x, y), line number first and then column number
					if i, err := strconv.Atoi(lineNumberString); err == nil {
						foundY := LineIndex(i - 1)
						e.redraw = e.GoTo(foundY, c, status)
						e.redrawCursor = e.redraw
						if x, err := strconv.Atoi(lineColumnString); err == nil { // no error
							foundX := x - 1
							tabs := strings.Count(e.Line(foundY), "\t")
							e.pos.sx = foundX + (tabs * (e.tabsSpaces.PerTab - 1))
							e.Center(c)
						}
					}

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "In " + baseErrorFilename + ": " + errorMessage, true, false, ""
					}
					return errorMessage, true, false, ""
				}
			} else if e.mode == mode.Lua {
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
						return "In " + baseErrorFilename + ": " + errorMessage, true, false, ""
					}
					return errorMessage, true, false, ""
				}
				break
			} else if e.mode == mode.Go && errorMarker == ":" && strings.Count(line, ":") >= 2 {
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
				break
			}
			prevLine = line
		}

		if e.mode == mode.Crystal {
			// Crystal has the location on a different line from the error message
			fields := strings.Split(crystalLocationLine, ":")
			if len(fields) != 3 {
				return errorMessage, true, false, ""
			}
			if y, err := strconv.Atoi(fields[1]); err == nil { // no error

				foundY := LineIndex(y - 1)
				e.redraw = e.GoTo(foundY, c, status)
				e.redrawCursor = e.redraw

				if x, err := strconv.Atoi(fields[2]); err == nil { // no error
					foundX := x - 1
					tabs := strings.Count(e.Line(foundY), "\t")
					e.pos.sx = foundX + (tabs * (e.tabsSpaces.PerTab - 1))
					e.Center(c)
				}

			}
			return errorMessage, true, false, ""
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
					return "In " + baseErrorFilename + ": " + strings.TrimSpace(fields[3]), true, false, ""
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
						e.pos.sx = foundX + (tabs * (e.tabsSpaces.PerTab - 1))
						e.Center(c)

						// Use the error message as the status message
						if len(fields) >= 4 {
							if ext != ".hs" {
								return strings.Join(fields[3:], " "), true, false, ""
							}
							return errorMessage, true, false, ""
						}
					}
				}
				return errorMessage, true, false, ""
			} else if (i > 0) && i < (len(lines)-1) {
				// Rust
				if msgLine := lines[i-1]; strings.Contains(line, " --> ") && strings.Count(line, ":") == 2 && strings.Count(msgLine, ":") >= 1 {
					errorFields := strings.SplitN(msgLine, ":", 2)                  // Already checked for 2 colons
					errorMessage := strings.TrimSpace(errorFields[1])               // There will always be 3 elements in errorFields, so [1] is fine
					locationFields := strings.SplitN(line, ":", 3)                  // Already checked for 2 colons in line
					filenameFields := strings.SplitN(locationFields[0], " --> ", 2) // [0] is fine, already checked for " ---> "
					errorFilename := strings.TrimSpace(filenameFields[1])           // [1] is fine
					if filename != errorFilename {
						return "Error in " + errorFilename + ": " + errorMessage, true, false, ""
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
							e.pos.sx = foundX + (tabs * (e.tabsSpaces.PerTab - 1))
							e.Center(c)
							// Use the error message as the status message
							if errorMessage != "" {
								return errorMessage, true, false, ""
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
	if e.mode == mode.Python {
		return "Syntax OK", true, true, ""
	}

	// Could not interpret the error message, return the last line of the output
	if err != nil && len(outputString) > 0 {
		outputLines := strings.Split(outputString, "\n")
		lastLine := outputLines[len(outputLines)-1]
		return "Error: " + lastLine, false, false, ""
	}

	if !exists(exeFirstName) && exists(baseDirName) {
		return "Success", true, true, baseDirName
	}
	return "Success", true, true, exeFirstName
}
