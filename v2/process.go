//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/xyproto/files"
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
	parentPID := os.Getppid()

	// Fast path for Linux, where /proc is always mounted.
	if isLinux {
		if parentPath, err := files.GetProcPath(parentPID, "exe"); err == nil {
			return filepath.Base(parentPath) == name
		}
	}

	// Portable fallback using ps (works on FreeBSD, macOS, OpenBSD, etc.)
	output, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(parentPID)).Output()
	if err != nil {
		return false
	}
	commandName := strings.TrimSpace(string(output))
	if commandName == "" {
		return false
	}
	if fields := strings.Fields(commandName); len(fields) > 0 {
		commandName = fields[0]
	}
	return filepath.Base(commandName) == name
}

// parentCommand returns the command line of the parent process or an empty string if an error occurs.
func parentCommand() string {
	parentPID := os.Getppid()

	// Fast path for Linux, where /proc is always mounted.
	if isLinux {
		if commandString, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", parentPID)); err == nil {
			return string(commandString)
		}
	}

	// Portable fallback using ps (works on FreeBSD, macOS, OpenBSD, etc.)
	output, err := exec.Command("ps", "-o", "command=", "-p", strconv.Itoa(parentPID)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
