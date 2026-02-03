//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/vt"
)

// quitExecShellCommand executes a shell command and exits
func quitExecShellCommand(tty *vt.TTY, workDir string, shellCommand string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt.Reset()
	vt.SetNoColor()
	vt.Clear()
	vt.Close()
	vt.ShowCursor(true)
	vt.SetXY(uint(0), uint(1))
	const shellExecutable = "/bin/sh"
	_ = os.Chdir(workDir)
	syscall.Exec(shellExecutable, []string{shellExecutable, "-c", shellCommand}, env.Environ())
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
	vt.ShowCursor(true)
	syscall.Exec(batExecutable, args, env.Environ())
	return nil // this is never reached
}

func quitToMan(tty *vt.TTY, workDir, nroffFilename string, w, h uint) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	vt.Close()
	vt.SetNoColor()
	vt.Clear()
	vt.Reset()
	if tty != nil {
		tty.Close()
	}

	if err := os.Chdir(workDir); err != nil {
		return err
	}

	editorExecutable := files.WhichCached(env.Str("EDITOR"))
	if editorExecutable == "" {
		return fmt.Errorf("could not find %s in PATH", env.Str("EDITOR"))
	}

	manExecutable := files.WhichCached("man")
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
	env.Set("MANPAGER", editorExecutable)

	env.Set("COLUMNS", strconv.Itoa(int(w)))
	env.Set("LINES", strconv.Itoa(int(h)))

	syscall.Exec(manExecutable, args, env.Environ())
	return nil
}

func quitToNroff(tty *vt.TTY, backupDirectory string, w, h uint) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	vt.Close()
	vt.SetNoColor()
	vt.Clear()
	vt.Reset()

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
	env.Set("ORBITON_SPACE") // sets ORBITON_SPACE to "1"

	env.Set("COLUMNS", strconv.Itoa(int(w)))
	env.Set("LINES", strconv.Itoa(int(h)))

	syscall.Exec(oExecutable, args, env.Environ())
	return nil
}
