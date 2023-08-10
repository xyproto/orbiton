//go:build freebsd || linux || netbsd || openbsd || solaris || dragonfly
// +build freebsd linux netbsd openbsd solaris dragonfly

package clip

import (
	"errors"
	"os"
	"os/exec"
)

const (
	xsel               = "xsel"
	xclip              = "xclip"
	wlcopy             = "wl-copy"
	wlpaste            = "wl-paste"
	termuxClipboardGet = "termux-clipboard-get"
	termuxClipboardSet = "termux-clipboard-set"
	powershellExe      = "powershell.exe"
	clipExe            = "clip.exe"
)

var (
	trimDos bool

	pasteCmdArgs []string
	copyCmdArgs  []string

	xselPasteArgs = []string{xsel, "--output", "--clipboard"}
	xselCopyArgs  = []string{xsel, "--input", "--clipboard"}

	xclipPasteArgs = []string{xclip, "-out", "-selection", "clipboard"}
	xclipCopyArgs  = []string{xclip, "-in", "-selection", "clipboard"}

	powershellExePasteArgs = []string{powershellExe, "Get-Clipboard"}
	clipExeCopyArgs        = []string{clipExe}

	wlpasteArgs = []string{wlpaste, "--no-newline"}
	wlcopyArgs  = []string{wlcopy}

	termuxPasteArgs = []string{termuxClipboardGet}
	termuxCopyArgs  = []string{termuxClipboardSet}

	errMissingCommands = errors.New("No clipboard utilities available. Please install xsel, xclip, wl-clipboard or Termux:API add-on for termux-clipboard-get/set.")

	initialized bool
)

func initialize() {
	initialized = true
	if WSL() {
		pasteCmdArgs = powershellExePasteArgs
		copyCmdArgs = clipExeCopyArgs
		trimDos = true

		if _, err := exec.LookPath(clipExe); err == nil {
			if _, err := exec.LookPath(powershellExe); err == nil {
				return
			}
		}
	}

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		pasteCmdArgs = wlpasteArgs
		copyCmdArgs = wlcopyArgs

		if _, err := exec.LookPath(wlcopy); err == nil {
			if _, err := exec.LookPath(wlpaste); err == nil {
				return
			}
		}
	}

	pasteCmdArgs = xclipPasteArgs
	copyCmdArgs = xclipCopyArgs

	if _, err := exec.LookPath(xclip); err == nil {
		return
	}

	pasteCmdArgs = xselPasteArgs
	copyCmdArgs = xselCopyArgs

	if _, err := exec.LookPath(xsel); err == nil {
		return
	}

	pasteCmdArgs = termuxPasteArgs
	copyCmdArgs = termuxCopyArgs

	if _, err := exec.LookPath(termuxClipboardSet); err == nil {
		if _, err := exec.LookPath(termuxClipboardGet); err == nil {
			return
		}
	}

	pasteCmdArgs = powershellExePasteArgs
	copyCmdArgs = clipExeCopyArgs
	trimDos = true

	if _, err := exec.LookPath(clipExe); err == nil {
		if _, err := exec.LookPath(powershellExe); err == nil {
			return
		}
	}

	Unsupported = true
}

func getPasteCommand(primary bool) *exec.Cmd {
	if !initialized {
		initialize()
	}
	if primary {
		pasteCmdArgs = pasteCmdArgs[:1]
	}
	return exec.Command(pasteCmdArgs[0], pasteCmdArgs[1:]...)
}

func getCopyCommand(primary bool) *exec.Cmd {
	if !initialized {
		initialize()
	}
	if primary {
		copyCmdArgs = copyCmdArgs[:1]
	}
	return exec.Command(copyCmdArgs[0], copyCmdArgs[1:]...)
}

func readAllBytes(primary bool) ([]byte, error) {
	if !initialized {
		initialize()
	}
	if Unsupported {
		return []byte{}, errMissingCommands
	}
	pasteCmd := getPasteCommand(primary)
	out, err := pasteCmd.Output()
	if err != nil {
		return []byte{}, errors.New("could not run: " + pasteCmd.String())
	}
	pasteCmd.Wait()
	pasteCmd.Stdout, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
	pasteCmd.Stderr, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
	return out, nil
}

func readAll(primary bool) (string, error) {
	if !initialized {
		initialize()
	}
	b, err := readAllBytes(primary)
	if err != nil {
		return "", err
	}
	result := string(b)
	if trimDos && len(result) > 1 {
		result = result[:len(result)-2]
	}
	return result, nil
}

func writeAllBytes(b []byte, primary bool) error {
	if !initialized {
		initialize()
	}
	if Unsupported {
		return errMissingCommands
	}
	copyCmd := getCopyCommand(primary)
	in, err := copyCmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := copyCmd.Start(); err != nil {
		return err
	}
	if _, err := in.Write(b); err != nil {
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	return copyCmd.Wait()
}

func writeAll(text string, primary bool) error {
	if !initialized {
		initialize()
	}
	return writeAllBytes([]byte(text), primary)
}

// WSL returns true if this is a WSL distro
func WSL() bool {
	// the official way to detect a WSL distro
	// ref: https://github.com/microsoft/WSL/issues/423
	cmd := exec.Command("/bin/sh", "-c", "cat /proc/version | grep -o Microsoft")
	_, err := cmd.Output()
	return err == nil
}
