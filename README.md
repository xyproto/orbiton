# red

`red` is a limited, but relatively small and fast text editor.

For a more feature complete editor that is also written in Go, check out [micro](https://github.com/zyedidia/micro).

## Screenshot

![screenshot](img/screenshot.png)

## Installation

You can install `red` with Go 1.11 or later (development version):

    go get -u github.com/xyproto/red

## Features and limitations

* Has syntax highlighting for Go code.
* Never asks before saving or quitting. Be careful.
* Random characters may appear on the screen when keys are pressed. Clear them with `ctrl-l`.
* Keys like `Home` and `End` are not even registered by the key handler (but `ctrl-a` and `ctrl-e` works).
* Will strip trailing whitespace.
* Can format Go code using `gofmt`.
* Can be used for drawing "ASCII graphics".
* `esc` can be used to toggle "writing mode" where the cursor is limited to the end of lines and "ASCII graphics mode".
* Can handle text that contains the tab character (`\t`).
* Expects utilities like `gofmt` to be in `/usr/bin`.
* Does not handle terminal resizing, yet.
* Must be given a filename at start.

## Known bugs

* Some letters can not be typed. Like `ø`.
* The undo functionality is currently broken.
* If a line contains a unicode character (like `ø`), the cursor positioning is wrong after that position.

## Hotkeys

* `ctrl-q` to quit
* `ctrl-s` to save
* `ctrl-h` to toggle syntax highlighting for Go code
* `ctrl-f` to format the current file with `go fmt`
* `ctrl-a` go to start of line, then start of text
* `ctrl-e` go to end of line
* `ctrl-p` to scroll up 10 lines
* `ctrl-n` to scroll down 10 lines
* `ctrl-l` to redraw the screen
* `ctrl-k` to delete characters to the end of the line, then delete the line
* `ctrl-g` to show cursor positions, current letter and word count
* `ctrl-d` to delete a single character
* `ctrl-t` to toggle insert mode
* `ctrl-z` to undo
* `ctrl-x' to cut the current line
* `ctrl-c' to copy the current line
* `ctrl-v' to paste the current line
* `ctrl-b` to bookmark the current position
* `ctrl-j` to jump to the bookmark
* `esc` to toggle between "text edit mode" and "ASCII graphics mode"

## Size

The `red` executable is **409k** if built with GCC 9.1 (for 64-bit Linux):

    go build -gccgoflags '-Os -s'

For comparison, it's **2.8M** when building with Go 1.13 and no particular build flags are given.

## General info

* Version: 1.2.4
* License: MIT
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
