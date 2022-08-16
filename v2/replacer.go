package main

import (
	"bytes"
	"strings"
)

// opinionatedStringReplacer is a Replacer that can be used for fixing
// nonbreaking spaces, annoying tildes, \r\n and \r
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

// opinionatedByteReplacer takes a slice of bytes and can be used for fixing
// nonbreaking spaces, annoying tildes, \r\n and \r
func opinionatedByteReplacer(data []byte) []byte {
	// Replace non-breaking space with regular space
	data = bytes.ReplaceAll(data, []byte{0xc2, 0xa0}, []byte{0x20})
	// Fix annoying tilde
	data = bytes.ReplaceAll(data, []byte{0xcc, 0x88}, []byte{'~'})
	// Replace DOS line endings with UNIX line endings
	data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
	// Replace any remaining \r characters with \n
	return bytes.ReplaceAll(data, []byte{'\r'}, []byte{'\n'})
}
