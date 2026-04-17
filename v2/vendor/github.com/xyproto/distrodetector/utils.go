package distrodetector

import (
	"os/exec"
	"strings"
)

// capitalize capitalizes a string
func capitalize(s string) string {
	switch len(s) {
	case 0:
		return ""
	case 1:
		return strings.ToUpper(s)
	default:
		return strings.ToUpper(string(s[0])) + s[1:]
	}
}

// containsDigit checks if a string contains at least one digit
func containsDigit(s string) bool {
	for _, l := range s {
		if l >= '0' && l <= '9' {
			return true
		}
	}
	return false
}

// Has returns the full path to the given executable, or the original string
func Has(executable string) bool {
	_, err := exec.LookPath(executable)
	return err == nil
}

// Run a shell command and return the output, or an empty string
func Run(shellCommand string) string {
	cmd := exec.Command("sh", "-c", shellCommand)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return string(stdoutStderr)
}
