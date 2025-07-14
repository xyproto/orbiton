package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParentIsMan(t *testing.T) {
	if parentProcessIs("man") {
		t.Error("Parent process is 'man'.")
	} else {
		t.Logf("Parent process is not 'man'.")
	}
}

func TestParentProcessIsSelf(t *testing.T) {
	ppid := os.Getppid()
	ppName, err := getProcPath(ppid, "exe")
	if err != nil {
		t.Skipf("Skipping: cannot get parent process path: %v", err)
	}
	ppBase := filepath.Base(ppName)
	if !parentProcessIs(ppBase) {
		t.Errorf("Expected parentProcessIs(%q) to return true", ppBase)
	}
}

func TestGetProcPathSelf(t *testing.T) {
	pid := os.Getpid()
	exe, err := getProcPath(pid, "exe")
	if err != nil {
		t.Fatalf("Failed to get exe path for self: %v", err)
	}
	if !strings.HasPrefix(exe, "/") {
		t.Errorf("Expected absolute path, got: %s", exe)
	}
}

func TestParentCommandNotEmpty(t *testing.T) {
	cmd := parentCommand()
	if cmd == "" {
		t.Skip("Skipping: parent command is empty (may be a system process)")
	}
	if !strings.Contains(cmd, "\x00") {
		t.Errorf("Expected null-separated cmdline, got: %q", cmd)
	}
}

func TestGetPIDFoundProcess(t *testing.T) {
	selfExe, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get own executable: %v", err)
	}
	base := filepath.Base(selfExe)
	pid, err := getPID(base)
	if err != nil {
		t.Errorf("Expected to find self by name %q, got error: %v", base, err)
	}
	if pid <= 0 {
		t.Errorf("Expected valid pid for %q, got %d", base, pid)
	}
}

func TestGetPIDNotFound(t *testing.T) {
	_, err := getPID("nonexistent_process_name_123456789")
	if err == nil {
		t.Error("Expected error for nonexistent process, got nil")
	}
}

func TestFoundProcess(t *testing.T) {
	// This assumes the current process is visible to /proc
	selfExe, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get own executable: %v", err)
	}
	base := filepath.Base(selfExe)
	if !foundProcess(base) {
		t.Errorf("Expected to find current process %q", base)
	}
}

func TestFoundProcessFalse(t *testing.T) {
	if foundProcess("definitely_not_a_real_process_654321") {
		t.Error("Expected foundProcess to return false for nonexistent process")
	}
}

func TestStopBackgroundProcesses(t *testing.T) {
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start sleep process: %v", err)
	}
	runPID.Store(int64(cmd.Process.Pid))

	if !stopBackgroundProcesses() {
		t.Error("Expected stopBackgroundProcesses to kill the process")
	}

	// Wait for process to be reaped
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(1 * time.Second):
		t.Error("Process did not exit within timeout after SIGKILL")
	case err := <-done:
		if err == nil {
			t.Log("Process killed successfully")
		} else if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == -1 {
			t.Log("Process exited with SIGKILL")
		} else {
			t.Errorf("Unexpected process exit error: %v", err)
		}
	}
}

func TestPkill(t *testing.T) {
	var cmds []*exec.Cmd
	for i := 0; i < 2; i++ {
		cmd := exec.Command("sleep", "10")
		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start sleep process: %v", err)
		}
		cmds = append(cmds, cmd)
	}

	// Give time for /proc to reflect
	time.Sleep(100 * time.Millisecond)

	killed, err := pkill("sleep")
	if err != nil {
		t.Errorf("Expected to kill sleep processes, got error: %v", err)
	}
	if killed < 2 {
		t.Errorf("Expected to kill at least 2 processes, killed: %d", killed)
	}

	// Reap processes
	for _, cmd := range cmds {
		done := make(chan error)
		go func(c *exec.Cmd) {
			done <- c.Wait()
		}(cmd)

		select {
		case <-time.After(1 * time.Second):
			t.Errorf("Process %d did not exit after SIGKILL", cmd.Process.Pid)
		case err := <-done:
			if err == nil {
				t.Logf("Process %d exited cleanly", cmd.Process.Pid)
			} else {
				t.Logf("Process %d exited with error: %v", cmd.Process.Pid, err)
			}
		}
	}
}
