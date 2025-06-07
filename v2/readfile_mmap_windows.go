//go:build windows
// +build windows

package main

import "fmt"

// mmapFile stub for Windows: always returns an error to fall back to ReadFile.
func mmapFile(filename string) ([]byte, func() error, error) {
	return nil, nil, fmt.Errorf("memory-mapping not supported on Windows")
}
