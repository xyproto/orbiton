//go:build !windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/xyproto/env/v2"
)

// runningAsInit reports whether this process is PID 1, the system init
// process. This is the case when the kernel is booted with, for example,
// init=/usr/bin/o on the kernel command line.
//
// When running as init, Orbiton must never exit (the kernel panics with
// "Attempted to kill init" if PID 1 terminates) and must take on the minimal
// init duties, such as reaping orphaned child processes.
func runningAsInit() bool {
	return os.Getpid() == 1
}

// reapZombies reaps orphaned child processes that have been re-parented to
// this process. As PID 1 nobody else will wait() for orphans, so without
// reaping they pile up as un-collectable zombies and eventually exhaust the
// process table.
//
// Caveat: Orbiton (and the file browser) also launch and wait for their own
// subprocesses via os/exec. This reaper may occasionally collect such a child
// before os/exec's own Wait does, in which case that Wait returns a harmless
// "waitid: no child processes" error. In practice os/exec is already blocked
// in wait4() for its specific child when the child exits, so it almost always
// wins the race; this reaper only mops up true orphans. The trade-off only
// applies while running as PID 1.
func reapZombies() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGCHLD)
	for range sigChan {
		// Reap every child that is ready, without blocking.
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
// It is used as a fallback when running as PID 1 and no usable terminal or
// file browser is available, so that the machine is still usable instead of
// the kernel panicking on a dead init.
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
