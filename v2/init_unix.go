//go:build !windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/xyproto/env/v2"
)

// runningAsInit reports whether this process is PID 1, as when the kernel is
// booted with init=/usr/bin/o. PID 1 must never exit (the kernel panics on a
// dead init) and is responsible for reaping orphaned child processes.
func runningAsInit() bool {
	return os.Getpid() == 1
}

// reapZombies reaps orphaned child processes re-parented to PID 1. Nobody else
// wait()s for orphans, so without this they pile up as zombies. Orbiton's own
// os/exec subprocesses are usually reaped by their own Wait, but if this loop
// wins the race that Wait returns a harmless "no child processes" error.
func reapZombies() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGCHLD)
	for range sigChan {
		// Reap every ready child without blocking
		for {
			var ws syscall.WaitStatus
			pid, err := syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
			if pid <= 0 || err != nil {
				break
			}
		}
	}
}

// runInitShell launches an interactive login shell and waits for it to exit.
// Used as a fallback when running as PID 1 and no usable terminal is available,
// to keep the machine usable instead of leaving a dead init.
func runInitShell() {
	shell := env.Str("SHELL", "/bin/sh")
	cmd := exec.Command(shell, "-l")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// initBrowseDir returns the directory that the file browser should open when
// running as PID 1: the current working directory, or "/" as a fallback.
func initBrowseDir() string {
	if wd, err := os.Getwd(); err == nil && wd != "" {
		return wd
	}
	return "/"
}
