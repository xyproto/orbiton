//go:build windows

package main

import "os"

func preserveOwnership(_ *os.File, _ os.FileInfo) bool {
	return true
}
