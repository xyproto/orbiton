//go:build !windows
// +build !windows

package main

import (
	"golang.org/x/sys/unix"
	"os"
)

// mmapFile attempts to memory-map the named file (for non-Windows platforms).
// Returns the mapped data slice, an unmap function, or an error.
func mmapFile(filename string) ([]byte, func() error, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	size := int(stat.Size())
	if size == 0 {
		// Empty file; nothing to map
		return []byte{}, func() error { return file.Close() }, nil
	}
	data, err := unix.Mmap(int(file.Fd()), 0, size, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	unmap := func() error {
		err1 := unix.Munmap(data)
		err2 := file.Close()
		if err1 != nil {
			return err1
		}
		return err2
	}
	return data, unmap, nil
}
