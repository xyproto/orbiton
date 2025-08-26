package main

import (
	"bytes"
	"os/exec"
)

func pbcopy(s string) error {
	cmd := exec.Command("pbcopy")
	var buf bytes.Buffer
	buf.WriteString(s)
	cmd.Stdin = &buf
	return cmd.Run()
}

func pbpaste() (string, error) {
	cmd := exec.Command("pbpaste")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
