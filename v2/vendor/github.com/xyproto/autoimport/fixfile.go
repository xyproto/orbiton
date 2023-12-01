package autoimport

import (
	"os"
	"strings"
)

// Fix reads in a file and tries to organize the imports.
// removeExistingImports can be set to true to remove existing imports
// deGlob can be set to true to try to expand wildcard imports
func Fix(filename string, removeExistingImports, deGlob, verbose bool) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	onlyJava := strings.HasSuffix(strings.ToLower(filename), ".java")
	ima, err := New(onlyJava, removeExistingImports, deGlob)
	if err != nil {
		return data, nil // no change
	}
	newData, err := ima.FixImports(data, verbose)
	if err != nil {
		return data, nil // no change
	}
	// with fixed imports
	return newData, nil
}
