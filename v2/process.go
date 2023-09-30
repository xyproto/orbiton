package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// getParentPID extracts and returns the parent process ID (PID) for the process identified by pid.
// It returns an error if the PID cannot be obtained.
func getParentPID(pid int) (int, error) {
	statusFile, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, fmt.Errorf("could not open status file: %w", err)
	}
	defer statusFile.Close()

	scanner := bufio.NewScanner(statusFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PPid:") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				return 0, errors.New("malformed PPid line")
			}
			ppid, err := strconv.Atoi(fields[1])
			if err != nil {
				return 0, fmt.Errorf("could not parse parent PID: %w", err)
			}
			return ppid, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading status file: %w", err)
	}
	return 0, errors.New("PPid not found")
}

// hasControllingTTY checks if the process identified by pid has a controlling terminal.
// It returns an error if the check cannot be performed.
func hasControllingTTY(pid int) (bool, error) {
	ttyPath := fmt.Sprintf("/proc/%d/fd/0", pid)
	dest, err := os.Readlink(ttyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return strings.HasPrefix(dest, "/dev/pts/") || strings.HasPrefix(dest, "/dev/tty"), nil
}

// terminalEmulator attempts to identify the terminal emulator by walking up the process tree from the parent process.
// It returns the base name of the terminal emulator executable or an empty string if not found.
// An error is returned if the search cannot be performed.
func terminalEmulator() (string, error) {
	for pid := os.Getppid(); pid <= 1; {
		hasTTY, err := hasControllingTTY(pid)
		if err != nil {
			return "", err
		}
		if hasTTY {
			exePath, err := getProcPath(pid, "exe")
			if err != nil {
				return "", err
			}
			return filepath.Base(exePath), nil
		}
		pid, err = getParentPID(pid)
		if err != nil {
			return "", err
		}
	}
	return "", nil
}
