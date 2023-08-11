package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// manIsParent checks if the parent process is an executable named "man"
func manIsParent() bool {
	parentPID := os.Getppid()
	parentPath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", parentPID))
	if err != nil {
		return false
	}
	baseName := filepath.Base(parentPath)
	return baseName == "man"
}

// parentCommand returns either the command of the parent process or an empty string
func parentCommand() string {
	commandString, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", os.Getppid()))
	if err != nil {
		return ""
	}
	return string(commandString)
}
