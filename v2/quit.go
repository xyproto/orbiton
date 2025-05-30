package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

func quitError(tty *vt100.TTY, err error) {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt100.Reset()
	vt100.SetNoColor()
	vt100.Clear()
	vt100.Close()
	textoutput.NewTextOutput(true, true).Err(err.Error())
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(1))
	os.Exit(1)
}

func quitMessage(tty *vt100.TTY, msg string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt100.Reset()
	vt100.SetNoColor()
	vt100.Clear()
	vt100.Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(newLineCount+1))
	os.Exit(1)
}

func quitMessageWithStack(tty *vt100.TTY, msg string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt100.Reset()
	vt100.SetNoColor()
	vt100.Clear()
	vt100.Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(newLineCount+1))
	debug.PrintStack()
	os.Exit(1)
}

func quitExecShellCommand(tty *vt100.TTY, workDir string, shellCommand string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt100.Reset()
	vt100.SetNoColor()
	vt100.Clear()
	vt100.Close()
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(1))
	const shellExecutable = "/bin/sh"
	_ = os.Chdir(workDir)
	syscall.Exec(shellExecutable, []string{shellExecutable, "-c", shellCommand}, env.Environ())
}

// CatBytes detects the source code mode and outputs syntax highlighted text to the given TextOutput.
func CatBytes(sourceCodeData []byte, o *textoutput.TextOutput) error {
	detectedMode := mode.SimpleDetectBytes(sourceCodeData)
	taggedTextBytes, err := AsText(sourceCodeData, detectedMode)
	if err == nil {
		o.OutputTags(string(taggedTextBytes))
	}
	return err
}

// quitCat tries to list the given source code file using CatBytes, and then exits
func quitCat(fnord *FilenameOrData) {
	quitMut.Lock()
	defer quitMut.Unlock()
	if fnord.Empty() {
		if sourceCodeBytes, err := os.ReadFile(fnord.filename); err == nil { // success
			if err := CatBytes(sourceCodeBytes, tout); err == nil { // success
				vt100.ShowCursor(true)
				os.Exit(0)
			}
		}
	} else {
		if err := CatBytes(fnord.data, tout); err == nil { // success
			vt100.ShowCursor(true)
			os.Exit(0)
		}
	}
	vt100.ShowCursor(true)
	os.Exit(1) // could not cat the file in a syntax highlighted way
}

// quitBat tries to list the given source code file using "bat", if "bat" exists in the path, and then exits
func quitBat(filename string) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	// ORBITON_BAT environment variable allows configuring the bat command and flags used when invoking "bat" via -c, -p, -b, or --bat options.
	batCommandLine := env.Str("ORBITON_BAT", "bat")
	batExecutable := batCommandLine
	args := []string{batExecutable}
	if strings.Contains(batCommandLine, " ") {
		batCommandLine = strings.ReplaceAll(batCommandLine, "\\ ", "\\")
		fields := strings.Split(batCommandLine, " ")
		batExecutable = files.Which(fields[0])
		args = append([]string{batExecutable}, fields[1:]...)
		for i, arg := range args {
			if strings.Contains(arg, "\\") {
				args[i] = strings.ReplaceAll(arg, "\\", " ")
			}
		}
	} else {
		batExecutable = files.Which(batExecutable)
	}
	if batExecutable == "" {
		return fmt.Errorf("%q is not available in the PATH", batExecutable)
	}
	args = append(args, filename)
	vt100.ShowCursor(true)
	syscall.Exec(batExecutable, args, env.Environ())
	return nil // this is never reached
}

func quitToMan(tty *vt100.TTY, workDir, nroffFilename string, w, h uint) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	vt100.Close()
	vt100.SetNoColor()
	vt100.Clear()
	vt100.Reset()
	if tty != nil {
		tty.Close()
	}

	if err := os.Chdir(workDir); err != nil {
		return err
	}

	oExecutable, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}

	manExecutable := files.Which("man")
	args := []string{manExecutable}

	if isLinux {
		args = append(args, "-l", nroffFilename)
	} else {
		absManPageFilename, err := filepath.Abs(nroffFilename)
		if err != nil {
			return err
		}
		args = append(args, absManPageFilename)
	}

	env.Set("NROFF_FILENAME", nroffFilename)
	env.Set("MANPAGER", oExecutable)

	env.Set("COLUMNS", strconv.Itoa(int(w)))
	env.Set("LINES", strconv.Itoa(int(h)))

	syscall.Exec(manExecutable, args, env.Environ())
	return nil
}

func quitToNroff(tty *vt100.TTY, backupDirectory string, w, h uint) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	vt100.Close()
	vt100.SetNoColor()
	vt100.Clear()
	vt100.Reset()

	if tty != nil {
		tty.Close()
	}

	oExecutable, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}

	args := []string{oExecutable, "-f"}

	if nroffFilename := env.Str("NROFF_FILENAME"); nroffFilename != "" {
		args = append(args, filepath.Base(nroffFilename))
		if dir := filepath.Dir(nroffFilename); dir != "" {
			if err := os.Chdir(filepath.Dir(nroffFilename)); err != nil {
				return err
			}
		} else {
			if err := os.Chdir(backupDirectory); err != nil {
				return err
			}
		}
	}

	env.Unset("NROFF_FILENAME")
	env.Unset("MANPAGER")
	env.Set("ORBITON_SPACE")

	env.Set("COLUMNS", strconv.Itoa(int(w)))
	env.Set("LINES", strconv.Itoa(int(h)))

	syscall.Exec(oExecutable, args, env.Environ())
	return nil
}
