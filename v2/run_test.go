package main

import (
	"testing"
)

func TestRun(t *testing.T) {
	err := run("ls /tmp")
	if err != nil {
		t.Fail()
	}
}
