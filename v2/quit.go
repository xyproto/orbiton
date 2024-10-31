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
)

func quitError(tty *TTY, err error) {
	quitMut.Lock()
	defer quitMut.Unlock()

	if tty != nil {
		tty.Close()
	}
	Reset()
	Clear()
	Close()
	NewTextOutput(true, true).Err(err.Error())
	ShowCursor(true)
	SetXY(uint(0), uint(1))
	os.Exit(1)
}

func quitMessage(tty *TTY, msg string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	if tty != nil {
		tty.Close()
	}
	Reset()
	Clear()
	Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	ShowCursor(true)
	SetXY(uint(0), uint(newLineCount+1))
	os.Exit(1)
}

func quitMessageWithStack(tty *TTY, msg string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	if tty != nil {
		tty.Close()
	}
	Reset()
	Clear()
	Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	ShowCursor(true)
	SetXY(uint(0), uint(newLineCount+1))
	debug.PrintStack()
	os.Exit(1)
}

func quitExecShellCommand(tty *TTY, workDir string, shellCommand string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	if tty != nil {
		tty.Close()
	}
	Reset()

	Clear()
	Close()
	ShowCursor(true)
	SetXY(uint(0), uint(1))
	const shellExecutable = "/bin/sh"
	_ = os.Chdir(workDir)
	syscall.Exec(shellExecutable, []string{shellExecutable, "-c", shellCommand}, env.Environ())
}

func quitToMan(tty *TTY, workDir, nroffFilename string, w, h uint) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	Close()
	Clear()
	Reset()
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

func quitToNroff(tty *TTY, backupDirectory string, w, h uint) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	Close()
	Clear()
	Reset()
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

	env.Set("COLUMNS", strconv.Itoa(int(w)))
	env.Set("LINES", strconv.Itoa(int(h)))

	syscall.Exec(oExecutable, args, env.Environ())
	return nil
}
