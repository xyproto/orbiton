package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
)

var runPID atomic.Int64

// stopBackgroundProcesses stops the "run" process that is running
// in the background, if runPID > 0. Returns true if something was killed.
func stopBackgroundProcesses() bool {
	if runPID.Load() <= 0 {
		return false // nothing was killed
	}
	// calling runPID.Load() twice, in case something happens concurrently after the first .Load()
	syscall.Kill(int(runPID.Load()), syscall.SIGKILL)
	runPID.Store(-1)
	return true // something was killed
}

// getProcPath resolves and returns the specified path (e.g., "exe", "cwd") for the process identified by pid.
// It returns an error if the path cannot be resolved.
func getProcPath(pid int, suffix string) (string, error) {
	return os.Readlink(fmt.Sprintf("/proc/%d/%s", pid, suffix))
}

// parentProcessIs checks if the parent process is an executable named the given string (such as "man").
func parentProcessIs(name string) bool {
	parentPath, err := getProcPath(os.Getppid(), "exe")
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

// getPID tries to find the PID, given a process name, similar to pgrep
func getPID(name string) (int64, error) {
	lowerName := strings.ToLower(name)
	procDir, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	for _, entry := range procDir {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		exePath, err := getProcPath(pid, "exe")
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(filepath.Base(exePath)), lowerName) {
			return int64(pid), nil
		}
	}
	return 0, os.ErrNotExist
}

// foundProcess returns true if a valid PID for the given process name is found in /proc, similar to how pgrep works
func foundProcess(name string) bool {
	pid, err := getPID(name)
	return err == nil && pid > 0
}

// pkill tries to find and kill all processes that match the given name.
// Returns the number of processes killed, and an error, if anything went wrong.
func pkill(name string) (int, error) {
	lowerName := strings.ToLower(name)
	procDir, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range procDir {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == os.Getpid() {
			continue
		}
		exePath, err := getProcPath(pid, "exe")
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(filepath.Base(exePath)), lowerName) {
			if err := syscall.Kill(pid, syscall.SIGKILL); err == nil {
				count++
			}
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("no process named %q found", name)
	}
	return count, nil
}
