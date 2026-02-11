//go:build !linux && !darwin && !freebsd && !netbsd && !windows && !plan9

package megafile

import "errors"

// uptime returns the current uptime in seconds
func uptime() (float64, error) {
	return 0, errors.New("unknown uptime system")
}
