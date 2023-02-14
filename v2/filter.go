package main

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/xyproto/mode"
)

// LoadClass attempts to convert a .class file with "jad" and then
// save the output as a new filename, so that .class files can be
// opened as the disassembled version.
func (e *Editor) LoadClass(filename string) ([]byte, error) {
	jadCommand := exec.Command("jad", "-b", "-dead", "-f", "-ff", "-o", "-p", "-s", "-space", ".decompiled.java", filename)
	saveCommand(jadCommand)
	output, err := jadCommand.Output() // ignore warnings on stderr
	if err != nil {
		return []byte{}, err
	}
	e.mode = mode.Java
	e.filename = filename[:len(filename)-len(".class")] + ".decompiled.java"

	// Remove "java.lang." qualifiers that are not needed
	data := bytes.ReplaceAll(output, []byte("java.lang."), []byte{})

	if err := os.WriteFile(e.filename, data, 0o644); err != nil {
		return data, err
	}
	return data, nil
}
