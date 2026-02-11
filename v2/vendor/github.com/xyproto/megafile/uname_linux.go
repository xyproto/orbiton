//go:build linux

package megafile

import (
	"bytes"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// utsnameFieldToString converts a syscall.Utsname field (which may be []int8
// or []uint8 depending on the architecture) to a Go string.
func utsnameFieldToString(p unsafe.Pointer, length int) string {
	b := unsafe.Slice((*byte)(p), length)
	if i := bytes.IndexByte(b, 0); i != -1 {
		return string(b[:i])
	}
	return string(b)
}

func uname() (string, string, string, error) {
	// Retrieve system information using uname()
	var unameData syscall.Utsname
	if err := syscall.Uname(&unameData); err != nil {
		return "", "", "", fmt.Errorf("failed to get system information (uname): %w", err)
	}
	hostname := utsnameFieldToString(unsafe.Pointer(&unameData.Nodename[0]), len(unameData.Nodename))
	if strings.Contains(hostname, ".") {
		fields := strings.SplitN(hostname, ".", 2)
		hostname = fields[0]
	}
	kernelRelease := utsnameFieldToString(unsafe.Pointer(&unameData.Release[0]), len(unameData.Release))
	machineArch := utsnameFieldToString(unsafe.Pointer(&unameData.Machine[0]), len(unameData.Machine))
	return hostname, kernelRelease, machineArch, nil
}
