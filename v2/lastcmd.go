package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var lastCommandFile = filepath.Join(userCacheDir, "o", "last_command.sh")

// tmpFileStripRegex and tmpFileStripOnce cache the regex for stripping /tmp/o.* suffixes.
var (
	tmpFileStripRegex *regexp.Regexp
	tmpFileStripOnce  sync.Once
)

// getCommand takes an *exec.Cmd and returns the command
// it represents, but with "/usr/bin/sh -c " trimmed away.
func getCommand(cmd *exec.Cmd) string {
	s := cmd.Path + " " + strings.Join(cmd.Args[1:], " ")
	return strings.TrimPrefix(s, "/usr/bin/sh -c ")
}

// Save the command as the "last command"
func saveCommand(cmd *exec.Cmd) error {
	if noWriteToCache {
		return nil
	}

	p := lastCommandFile

	// First create the folder for the lock file overview, if needed
	folderPath := filepath.Dir(p)
	_ = os.MkdirAll(folderPath, 0o755)

	// Prepare the file
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Strip the leading /usr/bin/sh -c command, if present
	commandString := getCommand(cmd)

	// Add environment variables, but only some
	selectedEnvVars := []string{"NO_COLOR", "PYTHON"}
	for _, assignment := range cmd.Env {
		if strings.Contains(assignment, "=") && containsSubstring(assignment, selectedEnvVars) {
			commandString = strings.Replace(assignment, "=", "=\"", 1) + "\" \\\n" + commandString
		}
	}

	// Write the contents, ignore the number of written bytes
	_, err = f.WriteString(fmt.Sprintf("#!/bin/sh\n%s\n", commandString))
	return err
}

// saveCommandString saves a command string as the "last command"
func saveCommandString(commandString string) error {
	if noWriteToCache {
		return nil
	}

	p := lastCommandFile

	folderPath := filepath.Dir(p)
	_ = os.MkdirAll(folderPath, 0o755)

	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("#!/bin/sh\n%s\n", commandString))
	return err
}

// Read last command tries to read the last used external command, but also present it in a nice way
func readLastCommand() (string, error) {
	data, err := os.ReadFile(lastCommandFile)
	if err != nil {
		return "", errors.New("no available last command")
	}
	// Remove the shebang
	firstLineAndRest := strings.SplitN(string(data), "\n", 2)
	if len(firstLineAndRest) != 2 || !strings.HasPrefix(firstLineAndRest[0], "#") {
		return "", fmt.Errorf("unrecognized contents in %s", lastCommandFile)
	}
	theRest := strings.TrimSpace(firstLineAndRest[1])
	// Strip out any /tmp/o.* suffix (cached regex compiled lazily)
	tmpFileStripOnce.Do(func() {
		tmpFileStripRegex = regexp.MustCompile(`/tmp/o\..*$`)
	})
	replaced := tmpFileStripRegex.ReplaceAllString(theRest, "")
	return replaced, nil
}
