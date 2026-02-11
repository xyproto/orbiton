//go:build plan9

package megafile

import (
	"os"
	"runtime"
	"strings"
)

func uname() (string, string, string, error) {
	hostname := "unknown"
	if data, err := os.ReadFile("/dev/sysname"); err == nil {
		if name := strings.TrimSpace(string(data)); name != "" {
			hostname = name
		}
	}
	return hostname, "plan9", runtime.GOARCH, nil
}
