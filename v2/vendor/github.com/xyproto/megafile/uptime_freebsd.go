//go:build freebsd

package megafile

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// uptime returns the current uptime in seconds
func uptime() (float64, error) {
	tv, err := unix.SysctlTimeval("kern.boottime")
	if err != nil {
		return 0, fmt.Errorf("failed to get system information (uptime): %w", err)
	}
	bootTime := time.Unix(tv.Sec, int64(tv.Usec)*1000)
	return time.Since(bootTime).Seconds(), nil
}
