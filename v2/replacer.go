package main

import (
	"strings"
)

// opinionatedStringReplacer is a Replacer that can be used for fixing:
// nonbreaking spaces, various Unicode spaces, annoying tildes, \r\n and \r,
// and other invisible characters that commonly cause issues in source code
var opinionatedStringReplacer = strings.NewReplacer(
	// Replace DOS line endings with UNIX line endings
	string([]byte{'\r', '\n'}), string([]byte{'\n'}),
	// Replace any remaining \r characters with \n
	string([]byte{'\r'}), string([]byte{'\n'}),
	// Replace non-breaking space with regular space
	string([]byte{0xc2, 0xa0}), string([]byte{0x20}),
	// Replace various Unicode space variants with regular space
	string([]byte{0xe2, 0x80, 0x89}), string([]byte{0x20}), // thin space
	string([]byte{0xe2, 0x80, 0xaf}), string([]byte{0x20}), // narrow no-break space
	string([]byte{0xe2, 0x80, 0x80}), string([]byte{0x20}), // en quad
	string([]byte{0xe2, 0x80, 0x81}), string([]byte{0x20}), // em quad
	string([]byte{0xe2, 0x80, 0x82}), string([]byte{0x20}), // en space
	string([]byte{0xe2, 0x80, 0x83}), string([]byte{0x20}), // em space
	string([]byte{0xe2, 0x80, 0x84}), string([]byte{0x20}), // three-per-em space
	string([]byte{0xe2, 0x80, 0x85}), string([]byte{0x20}), // four-per-em space
	string([]byte{0xe2, 0x80, 0x86}), string([]byte{0x20}), // six-per-em space
	string([]byte{0xe2, 0x80, 0x87}), string([]byte{0x20}), // figure space
	string([]byte{0xe2, 0x80, 0x88}), string([]byte{0x20}), // punctuation space
	string([]byte{0xe2, 0x80, 0x8a}), string([]byte{0x20}), // hair space
	string([]byte{0xe2, 0x81, 0x9f}), string([]byte{0x20}), // medium mathematical space
	// Fix greek question mark that looks like semicolon
	string([]byte{0xcd, 0xbe}), string([]byte{';'}),
	// Fix annoying tilde (combining diaeresis)
	string([]byte{0xcc, 0x88}), string([]byte{'~'}),
)
