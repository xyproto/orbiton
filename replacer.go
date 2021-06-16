package main

import (
	"bytes"
	"strings"
)

// Fix nonbreaking spaces, annoying tildes, \r\n and \r
var opinionatedStringReplacer = strings.NewReplacer(
	// Replace non-breaking space with regular space
	string([]byte{0xc2, 0xa0}), string([]byte{0x20}),
	// Fix annoying tilde
	string([]byte{0xcc, 0x88}), string([]byte{'~'}),
	// Replace DOS line endings with UNIX line endings
	string([]byte{'\r', '\n'}), string([]byte{'\n'}),
	// Replace any remaining \r characters with \n
	string([]byte{'\r'}), string([]byte{'\n'}),
)

// opinionatedByteReplacer will fix nonbreaking spaces, annoying tildes, \r\n and \r
func opinionatedByteReplacer(data []byte) []byte {
	// Replace non-breaking space with regular space
	data = bytes.Replace(data, []byte{0xc2, 0xa0}, []byte{0x20}, -1)
	// Fix annoying tilde
	data = bytes.Replace(data, []byte{0xcc, 0x88}, []byte{'~'}, -1)
	// Replace DOS line endings with UNIX line endings
	data = bytes.Replace(data, []byte{'\r', '\n'}, []byte{'\n'}, -1)
	// Replace any remaining \r characters with \n
	data = bytes.Replace(data, []byte{'\r'}, []byte{'\n'}, -1)
	return data
}
