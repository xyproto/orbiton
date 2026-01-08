//go:build linux

package megafile

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// uptime returns the current uptime in seconds
func uptime() (float64, error) {
	// Retrieve uptime information from /proc/uptime
	uptimeBytes, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("failed to open /proc/uptime: %w", err)
	}
	uptimeStr := strings.Fields(string(uptimeBytes))[0]
	uptimeSeconds, err := strconv.ParseFloat(uptimeStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uptime from /proc/uptime: %w", err)
	}
	return uptimeSeconds, nil
}
