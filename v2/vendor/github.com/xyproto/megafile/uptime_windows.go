//go:build windows

package megafile

import (
	"fmt"
	"syscall"
	"unsafe"
)

// uptime returns the current uptime in seconds
func uptime() (float64, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetTickCount64")
	ret, _, err := proc.Call()
	if err != nil && err.Error() != "The operation completed successfully." {
		return 0, fmt.Errorf("failed to get system information (uptime): %w", err)
	}
	_ = unsafe.Sizeof(ret)
	ms := uint64(ret)
	return float64(ms) / 1000.0, nil
}
