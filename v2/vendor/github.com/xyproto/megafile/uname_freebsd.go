//go:build freebsd

package megafile

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/sys/unix"
)

func uname() (string, string, string, error) {
	var unameData unix.Utsname
	if err := unix.Uname(&unameData); err != nil {
		return "", "", "", fmt.Errorf("failed to get system information (uname): %w", err)
	}
	hostname := string(bytes.TrimRight(unameData.Nodename[:], "\x00"))
	if strings.Contains(hostname, ".") {
		fields := strings.SplitN(hostname, ".", 2)
		hostname = fields[0]
	}
	kernelRelease := string(bytes.TrimRight(unameData.Release[:], "\x00"))
	machineArch := string(bytes.TrimRight(unameData.Machine[:], "\x00"))
	return hostname, kernelRelease, machineArch, nil
}
