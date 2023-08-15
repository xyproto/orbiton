package autoimport

import (
	"bytes"
	"fmt"
	"os"
)

// ForEachByteLine splits a file on '\n' and iterates over the byte slices
func ForEachByteLine(filename string, process func([]byte)) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("could not read %s: %v", filename, err)
	}
	byteLines := bytes.Split(data, []byte{'\n'})
	for _, byteLine := range byteLines {
		process(byteLine)
	}
	return nil
}

// ForEachLine splits a file on '\n' and iterates over the lines.
// The callback function will be given each line and trimmed line as the function iterates.
func ForEachLine(filename string, process func(string, string)) error {
	return ForEachByteLine(filename, func(byteLine []byte) {
		process(string(byteLine), string(bytes.TrimSpace(byteLine)))
	})
}

// FileImports generates sorted "import" lines for a .java or .kotlin file
// (the ImportMatcher should be configured to be either for Java or Kotlin as well)
func (ima *ImportMatcher) FileImports(filename string, verbose bool) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %v", filename, err)
	}
	importBlockBytes, err := ima.ImportBlock(data, verbose)
	if err != nil {
		return "", err
	}
	return string(importBlockBytes), nil
}
