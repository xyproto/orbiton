package main

import (
	"strings"

	"go4.org/bytereplacer"
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

// opinionatedByteReplacer is a bytereplacer that can be used for fixing
// nonbreaking spaces, annoying tildes, \r\n and \r
var opinionatedByteReplacer = bytereplacer.New(
	// Replace non-breaking space with regular space
	string([]byte{0xc2, 0xa0}), string([]byte{0x20}),
	// Fix annoying tilde
	string([]byte{0xcc, 0x88}), string([]byte{'~'}),
	// Replace DOS line endings with UNIX line endings
	string([]byte{'\r', '\n'}), string([]byte{'\n'}),
	// Replace any remaining \r characters with \n
	string([]byte{'\r'}), string([]byte{'\n'}),
)
