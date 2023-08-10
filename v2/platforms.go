package main

import (
	"runtime"
)

var (
	isDarwinCache *bool
	isLinuxCache  *bool
)

// isDarwin checks if the current OS is Darwin, and caches the result
func isDarwin() bool {
	if isDarwinCache != nil {
		return *isDarwinCache
	}
	b := runtime.GOOS == "darwin"
	isDarwinCache = &b
	return b
}

// isLinux checks if the current OS is Linux, and caches the result
func isLinux() bool {
	if isLinuxCache != nil {
		return *isLinuxCache
	}
	b := runtime.GOOS == "linux"
	isLinuxCache = &b
	return b
}
