//go:build !windows

package main

import (
	"fmt"
	"github.com/xyproto/files"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
)

var runPID atomic.Int64

// stopBackgroundProcesses stops the "run" process that is running
// in the background, if runPID > 0. Returns true if something was killed.
func stopBackgroundProcesses() bool {
	// Shutdown LSP clients
	ShutdownAllLSPClients()

	if runPID.Load() <= 0 {
		return false // nothing was killed
	}
	// calling runPID.Load() twice, in case something happens concurrently after the first .Load()
	syscall.Kill(int(runPID.Load()), syscall.SIGKILL)
	runPID.Store(-1)
	return true // something was killed
}

// parentProcessIs checks if the parent process is an executable named the given string (such as "man").
func parentProcessIs(name string) bool {
	parentPath, err := files.GetProcPath(os.Getppid(), "exe")
	if err != nil {
		return false
	}
	return err == nil && filepath.Base(parentPath) == name
}

// parentCommand returns the command line of the parent process or an empty string if an error occurs.
func parentCommand() string {
	if commandString, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", os.Getppid())); err == nil { // success
		return string(commandString)
	}
	return ""
}
