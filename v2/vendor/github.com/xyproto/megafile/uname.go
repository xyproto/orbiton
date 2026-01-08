//go:build !linux && !darwin

package megafile

import (
	"errors"
)

func uname() (string, string, string, error) {
	return "", "", "", errors.New("unknown uname system")
}
