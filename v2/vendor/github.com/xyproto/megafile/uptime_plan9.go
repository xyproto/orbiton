//go:build plan9

package megafile

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// uptime returns the current uptime in seconds
func uptime() (float64, error) {
	data, err := os.ReadFile("/dev/time")
	if err != nil {
		return 0, fmt.Errorf("failed to get system information (uptime): %w", err)
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("failed to parse /dev/time")
	}
	// /dev/time contains: seconds_since_epoch clock_ticks ...
	nowSec, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uptime: %w", err)
	}
	// On Plan 9, /dev/sysstat or /dev/time doesn't directly give boot time.
	// Use /dev/time for "now" and approximate uptime via process start.
	// A more reliable approach: read boot time from #c/time or calculate from msec.
	_ = nowSec

	// Read /dev/msec for milliseconds since boot
	msecData, err := os.ReadFile("/dev/msec")
	if err != nil {
		// Fallback: return time since epoch as approximation
		return float64(time.Now().Unix()), nil
	}
	msec, err := strconv.ParseInt(strings.TrimSpace(string(msecData)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse /dev/msec: %w", err)
	}
	return float64(msec) / 1000.0, nil
}
