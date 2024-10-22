package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

func quitError(tty *vt100.TTY, err error) {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	textoutput.NewTextOutput(true, true).Err(err.Error())
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(1))
	quitMut.Lock()
	defer quitMut.Unlock()
	os.Exit(1)
}

func quitMessage(tty *vt100.TTY, msg string) {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(newLineCount+1))
	quitMut.Lock()
	defer quitMut.Unlock()
	os.Exit(1)
}

func quitMessageWithStack(tty *vt100.TTY, msg string) {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	fmt.Fprintln(os.Stderr, msg)
	newLineCount := strings.Count(msg, "\n")
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(newLineCount+1))
	debug.PrintStack()
	quitMut.Lock()
	defer quitMut.Unlock()
	os.Exit(1)
}

func quitExecShellCommand(tty *vt100.TTY, workDir string, shellCommand string) {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(1))
	const shellExecutable = "/bin/sh"
	quitMut.Lock()
	defer quitMut.Unlock()
	_ = os.Chdir(workDir)
	syscall.Exec(shellExecutable, []string{shellExecutable, "-c", shellCommand}, env.Environ())
}

func quitExec(tty *vt100.TTY, workDir string, command string) {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(1))
	quitMut.Lock()
	defer quitMut.Unlock()
	_ = os.Chdir(workDir)

	executable := command
	args := []string{executable}
	if fields := strings.Split(command, " "); len(fields) > 0 {
		executable = fields[0] // TODO: Take abs path as well
		args = fields
	}

	env.Unset("CTRL_SPACE_COMMAND")

	syscall.Exec(executable, args, env.Environ())
}

func runMan(tty *vt100.TTY, workDir, manPageFilename string) error {
	if tty != nil {
		tty.Close()
	}
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	vt100.ShowCursor(true)
	vt100.SetXY(uint(0), uint(1))
	quitMut.Lock()
	defer quitMut.Unlock()
	_ = os.Chdir(workDir)
	oExecutable, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}

	manExecutable := files.Which("man")
	args := []string{manExecutable}

	if isLinux {
		args = append(args, "-E", "utf8", "-l", manPageFilename)
	} else {
		absManPageFilename, err := filepath.Abs(manPageFilename)
		if err != nil {
			return err
		}
		args = append(args, absManPageFilename)
	}

	env.Set("CTRL_SPACE_COMMAND", oExecutable+" "+manPageFilename)
	env.Set("MANPAGER", oExecutable)

	syscall.Exec(manExecutable, args, env.Environ())
	return nil
}
