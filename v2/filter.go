package main

import (
	"os/exec"

	"github.com/xyproto/mode"
)

// LoadClass attempts to convert a .class file with "cfr" and then
// save the output as a new filename, so that .class files can be
// opened as the disassembled version.
func (e *Editor) LoadClass(filename string) ([]byte, error) {
	e.mode = mode.Java
	e.filename = filename[:len(filename)-len(".class")] + ".decompiled.java"

	decompileCommand := exec.Command("cfr", filename, "--silent")
	saveCommand(decompileCommand)
	output, err := decompileCommand.Output() // ignore warnings on stderr
	if err != nil {
		return []byte{}, err
	}
	//if err := os.WriteFile(e.filename, output, 0o644); err != nil {
	//return []byte{}, err
	//}
	return output, nil
}
