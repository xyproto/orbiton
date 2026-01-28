//go:build linux

package megafile

import (
	"fmt"
	"strings"
	"syscall"
)

func uname() (string, string, string, error) {
	// Retrieve system information using uname()
	var unameData syscall.Utsname
	if err := syscall.Uname(&unameData); err != nil {
		return "", "", "", fmt.Errorf("failed to get system information (uname): %w", err)
	}
	hostname := trimNullBytes(unameData.Nodename[:])
	if strings.Contains(hostname, ".") {
		fields := strings.SplitN(hostname, ".", 2)
		hostname = fields[0]
	}
	kernelRelease := trimNullBytes(unameData.Release[:])
	machineArch := trimNullBytes(unameData.Machine[:])
	return hostname, kernelRelease, machineArch, nil
}
