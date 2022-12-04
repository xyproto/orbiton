package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

func (e *Editor) Run(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, filename string) (string, error) {
	sourceFilename, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	sourceDir := filepath.Dir(sourceFilename)
	var cmd *exec.Cmd

	switch e.mode {
	case mode.Kotlin:
		cmd = exec.Command("java", "-jar", strings.Replace(filename, ".kt", ".jar", 1))
		cmd.Dir = sourceDir
	default:
		return "", errors.New("run: not implemented for " + e.mode.String())
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
