//go:build darwin

package main

// isDarwin checks if the current OS is Darwin.
func isDarwin() bool {
	return true
}

// Other functions can be set to return false since they won't be relevant on Darwin
func isLinux() bool {
	return false
}

func isBSD() bool {
	return false
}
