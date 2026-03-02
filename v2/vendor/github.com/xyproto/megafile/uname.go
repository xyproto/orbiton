//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !windows && !plan9

package megafile

import (
	"errors"
)

func uname() (string, string, string, error) {
	return "", "", "", errors.New("unknown uname system")
}
