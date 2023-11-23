package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var lastCommandFile = filepath.Join(userCacheDir, "o", "last_command.sh")

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
	os.MkdirAll(folderPath, os.ModePerm)

	// Prepare the file
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Strip the leading /usr/bin/sh -c command, if present
	commandString := getCommand(cmd)

	// Write the contents, ignore the number of written bytes
	_, err = f.WriteString(fmt.Sprintf("#!/bin/sh\n%s\n", commandString))
	return err
}
