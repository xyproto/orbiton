// Package clip can be used for reading from and writing to the clipboard
package clip

// ReadAll will read a string from the clipboard
func ReadAll(primary bool) (string, error) {
	return readAll(primary)
}

// WriteAll will write a string to the clipboard
func WriteAll(text string, primary bool) error {
	return writeAll(text, primary)
}

// ReadAllBytes will read bytes from the clipboard
func ReadAllBytes(primary bool) ([]byte, error) {
	return readAllBytes(primary)
}

// WriteAllBytes will write bytes to the clipboard
func WriteAllBytes(b []byte, primary bool) error {
	return writeAllBytes(b, primary)
}

// Unsupported might be set true during clipboard init, to help callers decide
// whether or not to offer clipboard options.
var Unsupported bool
