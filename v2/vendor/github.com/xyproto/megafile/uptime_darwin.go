//go:build darwin

package megafile

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xyproto/files"
)

// uptime returns the current uptime in seconds
func uptime() (float64, error) {
	const startString = "sec = "
	const stopString = ","
	s, err := files.Shell("sysctl -n kern.boottime")
	if err != nil {
		return 0, fmt.Errorf("failed to get system information (uptime): %w", err)
	}
	//{ sec = 1766447163, usec = 441772 } Tue Dec 23 00:46:03 2025
	start := strings.Index(s, startString)
	if start == -1 {
		return 0, fmt.Errorf("could not parse uptime, could not find %q", startString)
	}
	start += len(startString)
	stop := strings.Index(s[start:], ",")
	if stop == -1 {
		return 0, fmt.Errorf("could not parse uptime, could not find %q", stopString)
	}
	stop += start
	bootTimeString := strings.TrimSpace(s[start:stop])
	bootTimeInt64, err := strconv.ParseInt(bootTimeString, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uptime: %w", err)
	}
	bootTime := time.Unix(bootTimeInt64, 0)
	return time.Since(bootTime).Seconds(), nil
}
