//go:build !windows

package main

import (
	"os"
	"syscall"
)

func preserveOwnership(f *os.File, fi os.FileInfo) bool {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return true
	}
	if st.Nlink > 1 {
		return false
	}
	if int(st.Uid) == os.Geteuid() && int(st.Gid) == os.Getegid() {
		return true
	}
	return f.Chown(int(st.Uid), int(st.Gid)) == nil
}
