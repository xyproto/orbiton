//go:build darwin
// +build darwin

package clip

import (
	"os/exec"
)

var (
	pasteCmdArgs = "pbpaste"
	copyCmdArgs  = "pbcopy"
)

func getPasteCommand() *exec.Cmd {
	return exec.Command(pasteCmdArgs)
}

func getCopyCommand() *exec.Cmd {
	return exec.Command(copyCmdArgs)
}

func readAll() (string, error) {
	pasteCmd := getPasteCommand()
	out, err := pasteCmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func readAllBytes() ([]byte, error) {
	pasteCmd := getPasteCommand()
	out, err := pasteCmd.Output()
	if err != nil {
		return []byte{}, err
	}
	return out, nil
}

func writeAll(text string) error {
	copyCmd := getCopyCommand()
	in, err := copyCmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := copyCmd.Start(); err != nil {
		return err
	}
	if _, err := in.Write([]byte(text)); err != nil {
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	return copyCmd.Wait()
}

func writeAllBytes(data []byte) error {
	copyCmd := getCopyCommand()
	in, err := copyCmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := copyCmd.Start(); err != nil {
		return err
	}
	if _, err := in.Write(data); err != nil {
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	return copyCmd.Wait()
}
