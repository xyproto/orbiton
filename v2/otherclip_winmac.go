//go:build !linux && !freebsd && !netbsd && !openbsd

package main

import "errors"

func getOtherClipboardContents() (string, error) {
	return "", errors.New("no \"other\" clipboard")
}
