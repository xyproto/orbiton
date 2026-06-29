package main

import (
	"strings"
)

// opinionatedStringReplacer is a Replacer that can be used for fixing:
// nonbreaking spaces, annoying tildes, \r\n and \r
var opinionatedStringReplacer = strings.NewReplacer(
	// Replace non-breaking space with regular space
	string([]byte{0xc2, 0xa0}), string([]byte{0x20}),
	// Fix annoying tilde
	string([]byte{0xcc, 0x88}), string([]byte{'~'}),
	// Fix greek question mark that looks like semicolon
	string([]byte{0xcd, 0xbe}), string([]byte{';'}),
	// Replace DOS line endings with UNIX line endings
	string([]byte{'\r', '\n'}), string([]byte{'\n'}),
	// Replace any remaining \r characters with \n
	string([]byte{'\r'}), string([]byte{'\n'}),
)

// pastedTextReplacer normalizes line endings and ambiguous runes in pasted
// text the same way InsertRune does when typing, so text inserted in bulk (a
// paste) gets the same treatment as text typed rune by rune. The rune code
// points match the substitutions done in InsertRune.
var pastedTextReplacer = strings.NewReplacer(
	"\r\n", "\n", // DOS line endings -> UNIX line endings
	"\r", "\n", // any remaining \r -> \n
	"\u00A0", " ", // non-breaking space -> regular space
	"\u0308", "~", // combining diaeresis (sticky dead key)
	"\u037E", ";", // Greek question mark (looks like ';')
	"\u0387", ";", // Greek ano teleia (looks like ';')
	"\u00B7", ".", // middle dot (looks like '.')
)
