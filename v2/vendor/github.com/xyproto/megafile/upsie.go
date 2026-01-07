package megafile

import (
	"bytes"
	"fmt"
	"io" // Added for error logging for pprof
	"os" // Added for CPU profiling
	"strconv"
	"strings"
	"syscall"
)

// trimNullBytes converts a null-terminated []int8 slice to a Go string.
// It explicitly copies the int8 values to a byte slice.
func trimNullBytes(s []int8) string {
	b := make([]byte, len(s))
	for i, v := range s {
		b[i] = byte(v)
	}
	// Find the first null byte and slice up to that point.
	if i := bytes.IndexByte(b, 0); i != -1 {
		b = b[:i]
	}
	return string(b)
}

// writeUptime formats and writes the uptime duration to the given writer.
func writeUptime(w io.Writer, totalSeconds int64) {
	if totalSeconds == 0 {
		fmt.Fprint(w, "just started")
		return
	}
	if totalSeconds < 60 {
		fmt.Fprint(w, "less than 1m")
		return
	}
	var b strings.Builder
	totalMinutes := totalSeconds / 60
	minutes := totalMinutes % 60
	totalHours := totalMinutes / 60
	hours := totalHours % 24
	totalDays := totalHours / 24
	days := totalDays % 7
	weeks := totalDays / 7
	if weeks > 0 {
		fmt.Fprintf(&b, "%dw", weeks)
	}
	if days > 0 {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%dd", days)
	}
	if hours > 0 {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%dh", hours)
	}
	if minutes > 0 {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%dm", minutes)
	}
	fmt.Fprint(w, b.String())
}

func upsieString(fullKernelVersion bool) (string, error) {
	// Retrieve system information using uname()
	var unameData syscall.Utsname
	if err := syscall.Uname(&unameData); err != nil {
		return "", fmt.Errorf("failed to get system information (uname): %w", err)
	}

	hostname := trimNullBytes(unameData.Nodename[:])
	kernelRelease := trimNullBytes(unameData.Release[:])
	machineArch := trimNullBytes(unameData.Machine[:])

	// Retrieve uptime information from /proc/uptime
	uptimeBytes, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/uptime: %w", err)
	}
	uptimeStr := strings.Fields(string(uptimeBytes))[0]
	uptimeSeconds, err := strconv.ParseFloat(uptimeStr, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse uptime from /proc/uptime: %w", err)
	}

	// Prepare kernel version string based on 'fullKernelVersion' flag
	kernelVersionDisplay := kernelRelease
	if !fullKernelVersion {
		parts := strings.Split(kernelRelease, ".")
		if len(parts) >= 2 {
			kernelVersionDisplay = parts[0] + "." + parts[1]
		}
	}

	var sb strings.Builder

	// Print the combined information.
	// Format: Hostname @ KernelVersion (Arch) - Up: <uptime_string>
	sb.WriteString(fmt.Sprintf(
		"%s%s%s %s@%s %s%s%s %s(%s%s%s%s%s)%s - %s%s%s ", // Corrected format string
		"<blue>", hostname, "</blue>", // Hostname
		"<white>", "</white>", // @
		"<red>", kernelVersionDisplay, "</red>", // Kernel
		"<darkgray>", "</darkgray>",
		"<darkyellow>", machineArch, "</darkyellow>", // Architecture
		"<darkgray>", "</darkgray>",
		"<yellow>", "Up:", "</yellow>", // "Up:" label and its colors
	))

	// Build and print uptime string
	sb.WriteString("<yellow>") // Apply yellow color for the uptime value
	writeUptime(&sb, int64(uptimeSeconds))
	sb.WriteString("</yellow>") // Reset color

	return sb.String(), nil
}
