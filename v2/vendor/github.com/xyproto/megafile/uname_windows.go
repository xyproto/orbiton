//go:build windows

package megafile

import (
	"os"
	"runtime"
)

func uname() (string, string, string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return hostname, "windows", runtime.GOARCH, nil
}
