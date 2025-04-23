package main

import (
	"testing"

	"github.com/ActiveState/termtest"
	//"github.com/stretchr/testify/suite"
)

func TestBash(t *testing.T) {
	opts := termtest.Options{
		CmdName: "/bin/bash",
	}
	cp, err := termtest.NewTest(t, opts)
	require.NoError(t, err, "create console process")
	defer cp.Close()

	cp.SendLine("echo hello world")
	cp.Expect("hello world")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}
