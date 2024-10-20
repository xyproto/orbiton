package main

import (
	"fmt"
	"os"
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

func execMan(tty *vt100.TTY, workDir, manPageFilename string) {
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
	env.Set("MANPAGER", os.Args[0])
	manExecutable := files.Which("man")
	// TODO: Do not use syscall.Exec here, but rather launch a child process
	syscall.Exec(manExecutable, []string{manExecutable, "-l", manPageFilename}, env.Environ())
}
