package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// parentIsMan checks if the parent process is an executable named "man".
func parentIsMan() bool {
	parentPath, err := getProcPath(os.Getppid(), "exe")
	if err != nil {
		return false
	}
	baseName := filepath.Base(parentPath)
	return baseName == "man"
}

// parentCommand returns the command line of the parent process or an empty string if an error occurs.
func parentCommand() string {
	commandString, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", os.Getppid()))
	if err != nil {
		return ""
	}
	return string(commandString)
}

// getProcPath resolves and returns the specified path (e.g., "exe", "cwd") for the process identified by pid.
// It returns an error if the path cannot be resolved.
func getProcPath(pid int, suffix string) (string, error) {
	path, err := os.Readlink(fmt.Sprintf("/proc/%d/%s", pid, suffix))
	if err != nil {
		return "", err
	}
	return path, nil
}
