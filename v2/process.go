package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// getProcPath resolves and returns the specified path (e.g., "exe", "cwd") for the process identified by pid.
// It returns an error if the path cannot be resolved.
func getProcPath(pid int, suffix string) (string, error) {
	path, err := os.Readlink(fmt.Sprintf("/proc/%d/%s", pid, suffix))
	if err != nil {
		return "", err
	}
	return path, nil
}

// parentProcessIs checks if the parent process is an executable named the given string (such as "man").
func parentProcessIs(name string) bool {
	parentPath, err := getProcPath(os.Getppid(), "exe")
	if err != nil {
		return false
	}
	baseName := filepath.Base(parentPath)
	return baseName == name
}

// parentCommand returns the command line of the parent process or an empty string if an error occurs.
func parentCommand() string {
	commandString, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", os.Getppid()))
	if err != nil {
		return ""
	}
	return string(commandString)
}

// getPID tries to find the PID, given a process name, similar to pgrep
func getPID(name string) (int64, error) {
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
		if strings.Contains(strings.ToLower(filepath.Base(exePath)), strings.ToLower(name)) {
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
