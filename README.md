# red

Just a tiny editor, using VT100 terminal codes.

For a more modern editor, also written in Go, look into [micro](https://github.com/zyedidia/micro).

## Features and limitations

* Has syntax highlighting for Go code.
* Can run `gofmt`.
* Can be used for drawing "ASCII graphics".
* The editor must be given a filename at start.
* The editor is always in "overwrite mode". Characters are never inserted so that other characters are moved around, except for `ctrl-d` for deleting a character.
* All trailing spaces are removed when saving, but a final newline is kept.
* `Esc` can be used to toggle "writing mode" where the cursor is limited to the end of lines and "ASCII drawing mode".
* Can handle text that contains the tab character (`\t`).
* Keys like `Home` and `End` are not even registered by the key handler (but `ctrl-a` and `ctrl-e` works).
* There is no undo.
* Expects utilities like `gofmt` to be in `/usr/bin`.

## Known bugs

* Letters that are not a-z, A-Z or simple punctuation may not be possible to type in.
* Lines longer than the terminal width are not handled correctly.
* Characters may appear on the screen when keys are pressed. Clear them with `ctrl-l`.
* Unicode characters may not be displayed correctly when loading a file.

## Hotkeys

* `ctrl-q` to quit
* `ctrl-s` to save (don't use this on files you care about!)
* `ctrl-h` to toggle syntax highlighting for Go code.
* `ctrl-f` to format the current file with `go fmt` (but not save the result).
* `ctrl-a` go to start of line
* `ctrl-e` go to end of line
* `ctrl-p` scroll up 10 lines
* `ctrl-n` scroll down 10 lines
* `ctrl-l` to redraw the screen
* `ctrl-k` to delete characters to the end of the line
* `ctrl-g` to show cursor positions, current letter and word count
* `ctrl-d` to delete a single character
* `esc` to toggle "text edit mode" and "ASCII graphics mode"
