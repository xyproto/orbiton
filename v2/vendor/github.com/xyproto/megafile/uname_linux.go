//go:build linux

package megafile

import (
	"fmt"
	"syscall"
)

func uname() (string, string, string, error) {
	// Retrieve system information using uname()
	var unameData syscall.Utsname
	if err := syscall.Uname(&unameData); err != nil {
		return "", "", "", fmt.Errorf("failed to get system information (uname): %w", err)
	}
	hostname := trimNullBytes(unameData.Nodename[:])
	kernelRelease := trimNullBytes(unameData.Release[:])
	machineArch := trimNullBytes(unameData.Machine[:])
	return hostname, kernelRelease, machineArch, nil
}
