package main

import (
	"bytes"
	"errors"
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
	zigCacheDir               = filepath.Join(userCacheDir, "o", "zig")
	pyCachePrefix             = filepath.Join(userCacheDir, "o", "python")
	pandocMutex               sync.RWMutex
)

// exeName tries to find a suitable name for the executable, given a source filename
// For instance, "main" or the name of the directory holding the source filename.
func (e *Editor) exeName(sourceFilename string) string {
	var (
		exeFirstName = "main" // The default name
		sourceDir    = filepath.Dir(sourceFilename)
		parentDir    = filepath.Clean(filepath.Join(sourceDir, ".."))
	)

	// Find a suitable default executable first name
	switch e.mode {
	case mode.Assembly, mode.Kotlin, mode.Lua, mode.OCaml, mode.Rust, mode.Terra, mode.Zig:
		sourceDirectoryName := filepath.Base(sourceDir)
		if sourceDirectoryName == "build" {
			return filepath.Base(parentDir)
		}
		return sourceDirectoryName
	}

	return exeFirstName
}

// GenerateBuildCommand will generate a command for building the given filename (or for displaying HTML)
// If there are no errors, a exec.Cmd is returned together with a function that can tell if the build
// produced an executable, together with the executable name,
func (e *Editor) GenerateBuildCommand(filename string) (*exec.Cmd, func() (bool, string), error) {
	var cmd *exec.Cmd

	everythingIsFine := func() (bool, string) {
		return true, "everything"
	}

	nothingIsFine := func() (bool, string) {
		return false, "nothing"
	}

	// Find the absolute path to the source file
	sourceFilename, err := filepath.Abs(filename)
	if err != nil {
		return cmd, nothingIsFine, err
	}

	// Set up a few basic variables about the given source file
	var (
		sourceDir      = filepath.Dir(sourceFilename)
		parentDir      = filepath.Clean(filepath.Join(sourceDir, ".."))
		grandParentDir = filepath.Clean(filepath.Join(sourceDir, "..", ".."))
		exeFirstName   = e.exeName(sourceFilename)
		exeFilename    = filepath.Join(sourceDir, exeFirstName)
		jarFilename    = exeFirstName + ".jar"
	)

	exeExists := func() (bool, string) {
		return exists(filepath.Join(sourceDir, exeFirstName)), exeFirstName
	}

	exeBaseNameOrMainExists := func() (bool, string) {
		if exists(filepath.Join(sourceDir, exeFirstName)) {
			return true, exeFirstName
		}
		if baseDirName := filepath.Base(sourceDir); exists(filepath.Join(sourceDir, baseDirName)) {
			return true, baseDirName
		}
		return exists(filepath.Join(sourceDir, "main")), "main"
	}

	exeOrMainExists := func() (bool, string) {
		if exists(filepath.Join(sourceDir, exeFirstName)) {
			return true, exeFirstName
		}
		return exists(filepath.Join(sourceDir, "main")), "main"
	}

	switch e.mode {
	case mode.Java: // build a .jar file
		javaShellCommand := "javaFiles=$(find . -type f -name '*.java'); for f in $javaFiles; do grep -q 'static void main' \"$f\" && mainJavaFile=\"$f\"; done; className=$(grep -oP '(?<=class )[A-Z]+[a-z,A-Z,0-9]*' \"$mainJavaFile\" | head -1); packageName=$(grep -oP '(?<=package )[a-z,A-Z,0-9,.]*' \"$mainJavaFile\" | head -1); if [[ $packageName != \"\" ]]; then packageName=\"$packageName.\"; fi; mkdir -p _o_build/META-INF; javac -d _o_build $javaFiles; cd _o_build; echo \"Main-Class: $packageName$className\" > META-INF/MANIFEST.MF; classFiles=$(find . -type f -name '*.class'); jar cmf META-INF/MANIFEST.MF ../" + jarFilename + " $classFiles; cd ..; rm -rf _o_build"
		cmd = exec.Command("sh", "-c", javaShellCommand)
		cmd.Dir = sourceDir
		return cmd, func() (bool, string) {
			return exists(filepath.Join(sourceDir, jarFilename)), jarFilename
		}, nil
	case mode.Scala:
		// For building a .jar file that can not be run with "java -jar main.jar" but with "scala main.jar": scalac -jar main.jar Hello.scala
		scalaShellCommand := "scalaFiles=$(find . -type f -name '*.scala'); for f in $scalaFiles; do grep -q 'def main' \"$f\" && mainScalaFile=\"$f\"; grep -q ' extends App ' \"$f\" && mainScalaFile=\"$f\"; done; objectName=$(grep -oP '(?<=object )[A-Z]+[a-z,A-Z,0-9]*' \"$mainScalaFile\" | head -1); packageName=$(grep -oP '(?<=package )[a-z,A-Z,0-9,.]*' \"$mainScalaFile\" | head -1); if [[ $packageName != \"\" ]]; then packageName=\"$packageName.\"; fi; mkdir -p _o_build/META-INF; scalac -d _o_build $scalaFiles; cd _o_build; echo -e \"Main-Class: $packageName$objectName\\nClass-Path: /usr/share/scala/lib/scala-library.jar\" > META-INF/MANIFEST.MF; classFiles=$(find . -type f -name '*.class'); jar cmf META-INF/MANIFEST.MF ../" + jarFilename + " $classFiles; cd ..; rm -rf _o_build"
		// Compile directly to jar with scalac if /usr/share/scala/lib/scala-library.jar is not found
		if !exists("/usr/share/scala/lib/scala-library.jar") {
			scalaShellCommand = "scalac -d run_with_scala.jar $(find . -type f -name '*.scala')"
		}
		cmd = exec.Command("sh", "-c", scalaShellCommand)
		cmd.Dir = sourceDir
		return cmd, func() (bool, string) {
			return exists(filepath.Join(sourceDir, jarFilename)), jarFilename
		}, nil
	case mode.Kotlin:
		if which("kotlinc-native") != "" {
			cmd = exec.Command("kotlinc-native", "-nowarn", "-opt", "-Xallocator=mimalloc", "-produce", "program", "-linker-option", "--as-needed", sourceFilename, "-o", exeFirstName)
			cmd.Dir = sourceDir
			return cmd, func() (bool, string) {
				if exists(filepath.Join(sourceDir, exeFirstName+".kexe")) {
					return true, exeFirstName + ".kexe"
				}
				return exists(filepath.Join(sourceDir, exeFirstName)), exeFirstName
			}, nil
		}
		cmd = exec.Command("kotlinc", sourceFilename, "-include-runtime", "-d", jarFilename)
		cmd.Dir = sourceDir
		return cmd, func() (bool, string) {
			return exists(filepath.Join(sourceDir, jarFilename)), jarFilename
		}, nil
	case mode.Go:
		cmd := exec.Command("go", "build")
		if strings.HasSuffix(sourceFilename, "_test.go") {
			// go test run a test that does not exist in order to build just the tests
			// thanks @cespare at github
			// https://github.com/golang/go/issues/15513#issuecomment-216410016
			cmd = exec.Command("go", "test", "-run", "xxxxxxx")
		}
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Hare:
		cmd := exec.Command("hare", "build")
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.C:
		if which("cxx") != "" {
			cmd = exec.Command("cxx")
			cmd.Dir = sourceDir
			if e.debugMode {
				cmd.Args = append(cmd.Args, "debugnosan")
			}
			return cmd, exeBaseNameOrMainExists, nil
		}
		// Use gcc directly
		if e.debugMode {
			cmd = exec.Command("gcc", "-o", exeFilename, "-Og", "-g", "-pipe", "-D_BSD_SOURCE", sourceFilename)
			cmd.Dir = sourceDir
			return cmd, exeExists, nil
		}
		cmd = exec.Command("gcc", "-o", exeFilename, "-O2", "-pipe", "-fPIC", "-fno-plt", "-fstack-protector-strong", "-D_BSD_SOURCE", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeExists, nil
	case mode.Cpp:
		if exists("BUILD.bazel") && which("bazel") != "" { // Google-style C++ + Bazel projects if
			return exec.Command("bazel", "build"), everythingIsFine, nil
		}
		if which("cxx") != "" {
			cmd = exec.Command("cxx")
			cmd.Dir = sourceDir
			if e.debugMode {
				cmd.Args = append(cmd.Args, "debugnosan")
			}
			return cmd, exeBaseNameOrMainExists, nil
		}
		// Use g++ directly
		if e.debugMode {
			cmd = exec.Command("g++", "-o", exeFilename, "-Og", "-g", "-pipe", "-Wall", "-Wshadow", "-Wpedantic", "-Wno-parentheses", "-Wfatal-errors", "-Wvla", "-Wignored-qualifiers", sourceFilename)
			cmd.Dir = sourceDir
			return cmd, exeExists, nil
		}
		cmd = exec.Command("g++", "-o", exeFilename, "-O2", "-pipe", "-fPIC", "-fno-plt", "-fstack-protector-strong", "-Wall", "-Wshadow", "-Wpedantic", "-Wno-parentheses", "-Wfatal-errors", "-Wvla", "-Wignored-qualifiers", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeExists, nil
	case mode.Zig:
		if which("zig") != "" {
			if exists("build.zig") {
				cmd = exec.Command("zig", "build")
				cmd.Dir = sourceDir
				return cmd, everythingIsFine, nil
			}
			// Just build the current file
			sourceCode := ""
			sourceData, err := os.ReadFile(sourceFilename)
			if err == nil { // success
				sourceCode = string(sourceData)
			}

			cmd = exec.Command("zig", "build-exe", "-lc", sourceFilename, "--name", exeFirstName, "--cache-dir", zigCacheDir)
			cmd.Dir = sourceDir
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
			return cmd, exeExists, nil
		}
		// No result
	case mode.V:
		cmd = exec.Command("v", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeOrMainExists, nil
	case mode.Garnet:
		cmd = exec.Command("garnetc", "-o", exeFirstName, sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeExists, nil
	case mode.Rust:
		if e.debugMode {
			cmd = exec.Command("cargo", "build", "--profile", "dev")
		} else {
			cmd = exec.Command("cargo", "build", "--profile", "release")
		}
		if exists("Cargo.toml") {
			cmd.Dir = sourceDir
			return cmd, everythingIsFine, nil
		}
		if exists(filepath.Join(parentDir, "Cargo.toml")) {
			cmd.Dir = parentDir
			return cmd, everythingIsFine, nil
		}
		// Use rustc instead of cargo if Cargo.toml is missing
		if rustcExecutable := which("rustc"); rustcExecutable != "" {
			if e.debugMode {
				cmd = exec.Command(rustcExecutable, sourceFilename, "-g", "-o", exeFilename)
			} else {
				cmd = exec.Command(rustcExecutable, sourceFilename, "-o", exeFilename)
			}
			cmd.Dir = sourceDir
			return cmd, exeExists, nil
		}
		// No result
	case mode.Clojure:
		cmd = exec.Command("lein", "uberjar")
		projectFileExists := exists("project.clj")
		parentProjectFileExists := exists("../project.clj")
		grandParentProjectFileExists := exists("../../project.clj")
		cmd.Dir = sourceDir
		if !projectFileExists && parentProjectFileExists {
			cmd.Dir = parentDir
		} else if !projectFileExists && !parentProjectFileExists && grandParentProjectFileExists {
			cmd.Dir = grandParentDir
		}
		return cmd, everythingIsFine, nil
	case mode.Haskell:
		cmd = exec.Command("ghc", "-dynamic", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Python:
		cmd = exec.Command("python", "-m", "py_compile", sourceFilename)
		cmd.Env = append(cmd.Env, "PYTHONUTF8=1")
		if !exists(pyCachePrefix) {
			os.MkdirAll(pyCachePrefix, 0700)
		}
		cmd.Env = append(cmd.Env, "PYTHONPYCACHEPREFIX="+pyCachePrefix)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.OCaml:
		cmd = exec.Command("ocamlopt", "-o", exeFirstName, sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeExists, nil
	case mode.Crystal:
		cmd = exec.Command("crystal", "build", "--no-color", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Erlang:
		cmd = exec.Command("erlc", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Lua:
		cmd = exec.Command("luac", "-o", exeFirstName+".out", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Nim:
		cmd = exec.Command("nim", "c", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.ObjectPascal:
		cmd = exec.Command("fpc", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.D:
		if e.debugMode {
			cmd = exec.Command("gdc", "-Og", "-g", "-o", exeFirstName, sourceFilename)
		} else {
			cmd = exec.Command("gdc", "-o", exeFirstName, sourceFilename)
		}
		cmd.Dir = sourceDir
		return cmd, exeExists, nil
	case mode.HTML:
		cmd = exec.Command("xdg-open", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Odin:
		cmd = exec.Command("odin", "build", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.CS:
		cmd = exec.Command("csc", "-nologo", "-unsafe", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.StandardML:
		cmd = exec.Command("mlton", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Agda:
		cmd = exec.Command("agda", "-c", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Assembly:
		objFullFilename := exeFilename + ".o"
		objCheckFunc := func() (bool, string) {
			// Note that returning the full path as the second argument instead of only the base name
			// is only done for mode.Assembly. It's treated differently further down when linking.
			return exists(objFullFilename), objFullFilename
		}
		// try to use yasm
		if which("yasm") != "" {
			cmd = exec.Command("yasm", "-f", "elf64", "-o", objFullFilename, sourceFilename)
			if e.debugMode {
				cmd.Args = append(cmd.Args, "-g", "dwarf2")
			}
			return cmd, objCheckFunc, nil
		}
		// then try to use nasm
		if which("nasm") != "" { // use nasm
			cmd = exec.Command("nasm", "-f", "elf64", "-o", objFullFilename, sourceFilename)
			if e.debugMode {
				cmd.Args = append(cmd.Args, "-g")
			}
			return cmd, objCheckFunc, nil
		}
		// No result
	}
	return nil, nothingIsFine, errNoSuitableBuildCommand //errors.New("No build command for " + e.mode.String() + " files")
}

// BuildOrExport will try to build the source code or export the document.
// Returns a status message and then true if an action was performed and another true if compilation/testing worked out.
// Will also return the executable output file, if available after compilation.
func (e *Editor) BuildOrExport(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, filename string, background bool) (string, error) {
	// Clear the status messages, if we have a status bar
	if status != nil {
		status.ClearAll(c)
	}

	// Find the absolute path to the source file
	sourceFilename, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}

	// Set up a few basic variables about the given source file
	var (
		baseFilename = filepath.Base(sourceFilename)
		sourceDir    = filepath.Dir(sourceFilename)
		exeFirstName = e.exeName(sourceFilename)
		exeFilename  = filepath.Join(sourceDir, exeFirstName)
		ext          = filepath.Ext(sourceFilename)
	)

	// Get a few simple cases out of the way first, by filename extension
	switch ext {
	case ".scd", ".scdoc": // scdoc
		manFilename := "out.1"
		if err := e.exportScdoc(manFilename); err != nil {
			return "", err
		}
		if status != nil {
			status.SetMessage("Saved " + manFilename)
		}
		return manFilename, nil
	case ".adoc": // asciidoctor
		manFilename := "out.1"
		if err := e.exportAdoc(c, tty, manFilename); err != nil {
			return "", err
		}
		if status != nil {
			status.SetMessage("Saved " + manFilename)
		}
		return manFilename, nil
	}

	// Get a few simple cases out of the way first, by editor mode
	switch e.mode {
	case mode.Markdown, mode.Doc:
		// pandoc
		if pandocPath := which("pandoc"); pandocPath != "" {
			pdfFilename := strings.ReplaceAll(filepath.Base(sourceFilename), ".", "_") + ".pdf"
			if background {
				go func() {
					pandocMutex.Lock()
					_ = e.exportPandoc(c, tty, status, pandocPath, pdfFilename)
					pandocMutex.Unlock()
				}()
			} else {
				_ = e.exportPandoc(c, tty, status, pandocPath, pdfFilename)
			}
			// the exportPandoc function handles it's own status output
			return pdfFilename, nil
		}
		return "", errors.New("could not find pandoc")
	}

	// The immediate builds are done, time to build a exec.Cmd, run it and analyze the output

	cmd, compilationProducedSomething, err := e.GenerateBuildCommand(sourceFilename)
	if err != nil {
		return "", err
	}

	// Check that the resulting cmd.Path executable exists
	if which(cmd.Path) == "" {
		return "", errNoSuitableBuildCommand
	}

	// Display a status message with no timeout, about what is currently being done
	if status != nil {
		var progressStatusMessage string
		if e.mode == mode.HTML || e.mode == mode.XML {
			progressStatusMessage = "Displaying"
		} else if !e.debugMode {
			progressStatusMessage = "Building"
		}
		status.SetMessage(progressStatusMessage)
		status.ShowNoTimeout(c, e)
	}

	// Save the command in a temporary file
	saveCommand(cmd)

	// --- Compilation ---

	// Run the command and fetch the combined output from stderr and stdout.
	// Ignore the status code / error, only look at the output.
	output, err := cmd.CombinedOutput()

	// Done building, clear the "Building" message
	if status != nil {
		status.ClearAll(c)
	}

	// Get the exit code and combined output of the build command
	exitCode := 0
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode = exitError.ExitCode()
	}
	outputString := string(bytes.TrimSpace(output))

	// Check if there was a non-zero exit code together with no output
	if exitCode != 0 && len(outputString) == 0 {
		return "", errors.New("non-zero exit code and no error message")
	}

	// Also perform linking, if needed
	if ok, objFullFilename := compilationProducedSomething(); e.mode == mode.Assembly && ok {
		linkerCmd := exec.Command("ld", "-o", exeFilename, objFullFilename)
		linkerCmd.Dir = sourceDir
		if e.debugMode {
			linkerCmd.Args = append(linkerCmd.Args, "-g")
		}
		var linkerOutput []byte
		linkerOutput, err = linkerCmd.CombinedOutput()
		if err != nil {
			output = append(output, '\n')
			output = append(output, linkerOutput...)
		} else {
			os.Remove(objFullFilename)
		}
		// Replace the result check function
		compilationProducedSomething = func() (bool, string) {
			return exists(exeFilename), exeFirstName
		}
	}

	if usingKotlinNative := strings.HasSuffix(cmd.Path, "kotlinc-native"); usingKotlinNative && exists(exeFirstName+".kexe") {
		//panic("rename " + exeFirstName + ".kexe" + " -> " + exeFirstName)
		os.Rename(exeFirstName+".kexe", exeFirstName)
	}

	// NOTE: Don't do anything with the output and err variables here, let the if below handle it.

	errorMarker := "error:"
	if e.mode == mode.Crystal || e.mode == mode.ObjectPascal || e.mode == mode.StandardML || e.mode == mode.Python {
		errorMarker = "Error:"
	} else if e.mode == mode.CS {
		errorMarker = ": error "
	} else if e.mode == mode.Agda {
		errorMarker = ","
	}

	// Check if the error marker should be changed

	if e.mode == mode.Zig && bytes.Contains(output, []byte("nrecognized glibc version")) {
		byteLines := bytes.Split(output, []byte("\n"))
		fields := strings.Split(string(byteLines[0]), ":")
		errorMessage := "Error: unrecognized glibc version"
		if len(fields) > 1 {
			errorMessage += ": " + strings.TrimSpace(fields[1])
		}
		return "", errors.New(errorMessage)
	} else if e.mode == mode.Go {
		switch {
		case bytes.Contains(output, []byte(": undefined")):
			errorMarker = "undefined"
		case bytes.Contains(output, []byte(": error")):
			errorMarker = "error"
		case bytes.Contains(output, []byte("go: cannot find main module")):
			errorMessage := "no main module, try go mod init"
			return "", errors.New(errorMessage)
		case bytes.Contains(output, []byte("go: ")):
			byteLines := bytes.SplitN(output[4:], []byte("\n"), 2)
			return "", errors.New(string(byteLines[0]))
		case bytes.Count(output, []byte(":")) >= 2:
			errorMarker = ":"
		}
	} else if e.mode == mode.Odin {
		switch {
		case bytes.Contains(output, []byte(") ")):
			errorMarker = ") "
		}
	} else if exitCode == 0 && (e.mode == mode.HTML || e.mode == mode.XML) {
		return "", nil
	}

	// Did the command return a non-zero status code, or does the output contain "error:"?
	if err != nil || bytes.Contains(output, []byte(errorMarker)) { // failed tests also end up here

		// This is not for Go, since the word "error:" may not appear when there are errors

		errorMessage := "Build error"

		if e.mode == mode.Python {
			if errorLine, errorColumn, errorMessage := ParsePythonError(string(output), filepath.Base(filename)); errorLine != -1 {
				ignoreIndentation := true
				e.MoveToLineColumnNumber(c, status, errorLine, errorColumn, ignoreIndentation)
				return "", errors.New(errorMessage)
			}
			// This should never happen, the error message should be handled by ParsePythonError!
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			lastLine := lines[len(lines)-1]
			return "", errors.New(lastLine)
		} else if e.mode == mode.Agda {
			lines := strings.Split(string(output), "\n")
			if len(lines) >= 4 {
				fileAndLocation := lines[1]
				errorMessage := strings.TrimSpace(lines[2]) + " " + strings.TrimSpace(lines[3])
				if strings.Contains(fileAndLocation, ":") && strings.Contains(fileAndLocation, ",") && strings.Contains(fileAndLocation, "-") {
					fields := strings.SplitN(fileAndLocation, ":", 2)
					//filename := fields[0]
					lineAndCol := fields[1]
					fields = strings.SplitN(lineAndCol, ",", 2)
					lineNumberString := fields[0] // not index
					colRange := fields[1]
					fields = strings.SplitN(colRange, "-", 2)
					lineColumnString := fields[0] // not index

					e.MoveToNumber(c, status, lineNumberString, lineColumnString)

					return "", errors.New(errorMessage)
				}
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
									e.redraw, _ = e.GoTo(foundY, c, status)
									e.redrawCursor = e.redraw
									if x, err := strconv.Atoi(lineColumnString); err == nil { // no error
										foundX := x - 1
										tabs := strings.Count(e.Line(foundY), "\t")
										e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
										e.Center(c)
									}
								}
								return "", errors.New(errorMessage)
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
			} else if e.mode == mode.Hare {
				errorMessage = ""
				if strings.Contains(line, errorMarker) && strings.Contains(line, " at ") {
					descriptionFields := strings.SplitN(line, " at ", 2)
					errorMessage = descriptionFields[0]
					if strings.Contains(errorMessage, "error:") {
						fields := strings.SplitN(errorMessage, "error:", 2)
						errorMessage = fields[1]
					}
					filenameAndError := descriptionFields[1]
					filenameAndLoc := ""
					if strings.Contains(filenameAndError, ", ") {
						fields := strings.SplitN(filenameAndError, ", ", 2)
						filenameAndLoc = fields[0]
						errorMessage += ": " + fields[1]
					}
					fields := strings.SplitN(filenameAndLoc, ":", 3)
					errorFilename := fields[0]
					baseErrorFilename := filepath.Base(errorFilename)
					lineNumberString := fields[1]
					lineColumnString := fields[2]

					e.MoveToNumber(c, status, lineNumberString, lineColumnString)

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "", errors.New("in " + baseErrorFilename + ": " + errorMessage)
					}
					return "", errors.New(errorMessage)

				} else if strings.HasPrefix(line, "Error ") {
					fields := strings.Split(line[6:], ":")
					if len(fields) >= 4 {
						errorFilename := fields[0]
						baseErrorFilename := filepath.Base(errorFilename)
						lineNumberString := fields[1]
						lineColumnString := fields[2]
						errorMessage := fields[3]

						e.MoveToNumber(c, status, lineNumberString, lineColumnString)

						// Return the error message
						if baseErrorFilename != baseFilename {
							return "", errors.New("in " + baseErrorFilename + ": " + errorMessage)
						}
						return "", errors.New(errorMessage)
					}
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

					e.MoveToIndex(c, status, lineNumberString, lineColumnString)

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "", errors.New("in " + baseErrorFilename + ": " + errorMessage)
					}
					return "", errors.New(errorMessage)
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
						e.redraw, _ = e.GoTo(foundY, c, status)
						e.redrawCursor = e.redraw
						if x, err := strconv.Atoi(lineColumnString); err == nil { // no error
							foundX := x - 1
							tabs := strings.Count(e.Line(foundY), "\t")
							e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
							e.Center(c)
						}
					}

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "", errors.New("In " + baseErrorFilename + ": " + errorMessage)
					}
					return "", errors.New(errorMessage)
				}
			} else if e.mode == mode.Lua {
				if strings.Contains(line, " error near ") && strings.Count(line, ":") >= 3 {
					parts := strings.SplitN(line, ":", 4)
					errorMessage = parts[3]

					if i, err := strconv.Atoi(parts[2]); err == nil {
						foundY := LineIndex(i - 1)
						e.redraw, _ = e.GoTo(foundY, c, status)
						e.redrawCursor = e.redraw
					}

					baseErrorFilename := filepath.Base(parts[1])
					if baseErrorFilename != baseFilename {
						return "", errors.New("In " + baseErrorFilename + ": " + errorMessage)
					}
					return "", errors.New(errorMessage)
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
				return "", errors.New(errorMessage)
			}
			if y, err := strconv.Atoi(fields[1]); err == nil { // no error

				foundY := LineIndex(y - 1)
				e.redraw, _ = e.GoTo(foundY, c, status)
				e.redrawCursor = e.redraw

				if x, err := strconv.Atoi(fields[2]); err == nil { // no error
					foundX := x - 1
					tabs := strings.Count(e.Line(foundY), "\t")
					e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
					e.Center(c)
				}

			}
			return "", errors.New(errorMessage)
		}

		// NOTE: Don't return here even if errorMessage contains an error message

		// Analyze all lines
		for i, line := range lines {
			// Go, C++, Haskell, Kotlin and more
			if strings.Contains(line, "fatal error") {
				return "", errors.New(line)
			}
			if strings.Count(line, ":") >= 3 && (strings.Contains(line, "error:") || strings.Contains(line, errorMarker)) {
				fields := strings.SplitN(line, ":", 4)
				baseErrorFilename := filepath.Base(fields[0])
				// Check if the filenames are matching, or if the error is in a different file
				if baseErrorFilename != baseFilename {
					return "", errors.New("In " + baseErrorFilename + ": " + strings.TrimSpace(fields[3]))
				}
				// Go to Y:X, if available
				var foundY LineIndex
				if y, err := strconv.Atoi(fields[1]); err == nil { // no error
					foundY = LineIndex(y - 1)
					e.redraw, _ = e.GoTo(foundY, c, status)
					e.redrawCursor = e.redraw
					foundX := -1
					if x, err := strconv.Atoi(fields[2]); err == nil { // no error
						foundX = x - 1
					}
					if foundX != -1 {

						tabs := strings.Count(e.Line(foundY), "\t")
						e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
						e.Center(c)

						// Use the error message as the status message
						if len(fields) >= 4 {
							if ext != ".hs" {
								return "", errors.New(strings.Join(fields[3:], " "))
							}
							return "", errors.New(errorMessage)
						}
					}
				}
				return "", errors.New(errorMessage)
			} else if (i > 0) && i < (len(lines)-1) {
				// Rust
				if msgLine := lines[i-1]; strings.Contains(line, " --> ") && strings.Count(line, ":") == 2 && strings.Count(msgLine, ":") >= 1 {
					errorFields := strings.SplitN(msgLine, ":", 2)                  // Already checked for 2 colons
					errorMessage := strings.TrimSpace(errorFields[1])               // There will always be 3 elements in errorFields, so [1] is fine
					locationFields := strings.SplitN(line, ":", 3)                  // Already checked for 2 colons in line
					filenameFields := strings.SplitN(locationFields[0], " --> ", 2) // [0] is fine, already checked for " ---> "
					errorFilename := strings.TrimSpace(filenameFields[1])           // [1] is fine
					if filename != errorFilename {
						return "", errors.New("In " + errorFilename + ": " + errorMessage)
					}
					errorY := locationFields[1]
					errorX := locationFields[2]

					// Go to Y:X, if available
					var foundY LineIndex
					if y, err := strconv.Atoi(errorY); err == nil { // no error
						foundY = LineIndex(y - 1)
						e.redraw, _ = e.GoTo(foundY, c, status)
						e.redrawCursor = e.redraw
						foundX := -1
						if x, err := strconv.Atoi(errorX); err == nil { // no error
							foundX = x - 1
						}
						if foundX != -1 {
							tabs := strings.Count(e.Line(foundY), "\t")
							e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
							e.Center(c)
							// Use the error message as the status message
							if errorMessage != "" {
								return "", errors.New(errorMessage)
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
		if status != nil {
			status.SetMessage("Syntax OK")
			status.Show(c, e)
		}
		return "", nil
	}

	// Could not interpret the error message, return the last line of the output
	if exitCode != 0 && len(outputString) > 0 {
		outputLines := strings.Split(outputString, "\n")
		lastLine := outputLines[len(outputLines)-1]
		return "", errors.New(lastLine)
	}

	if ok, what := compilationProducedSomething(); ok {
		// Returns the built executable, or exported file
		return what, nil
	}

	// TODO: Find ways to make the error message more informative
	return "", errors.New("could not compile")
}
