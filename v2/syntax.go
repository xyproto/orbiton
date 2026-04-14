package main

import "github.com/xyproto/syntax"

// SingleLineCommentMarker will return the string that starts a single-line
// comment for the current language mode the editor is in.
func (e *Editor) SingleLineCommentMarker() string {
	return syntax.SingleLineCommentMarker(e.mode)
}
