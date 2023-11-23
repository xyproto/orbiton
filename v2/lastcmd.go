package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var lastCommandFile = filepath.Join(userCacheDir, "o", "last_command.sh")

// isProllyFilename checks if the argument is likely a filename based on the presence
// of an OS-specific path separator and a ".".
func isProllyFilename(arg string) bool {
	return strings.ContainsRune(arg, os.PathSeparator) && strings.Contains(arg, ".")
}

// getCommand takes an *exec.Cmd and returns the command
// it represents, but with "/usr/bin/sh -c " trimmed away
// and filenames quoted.
func getCommand(cmd *exec.Cmd) string {
	var args []string
	for _, arg := range cmd.Args[1:] {
		if isProllyFilename(arg) {
			// Quote what appears to be a filename (has / and .)
			args = append(args, fmt.Sprintf("%q", arg))
		} else {
			args = append(args, arg)
		}
	}
	s := cmd.Path + " " + strings.Join(args, " ")
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
