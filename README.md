# o

`o` is a limited, but small and fast text editor.

Compiles with either `go` or `gccgo`. Tested with `st`, `urxvt` and `xfce4-terminal`.

For a more feature complete editor that is also written in Go, check out [micro](https://github.com/zyedidia/micro).

## Screenshot

![screenshot](img/screenshot.png)

## Quick Start

You can install `o` with Go 1.11 or later (development version):

    go get -u github.com/xyproto/o

## Features and limitations

* Has syntax highlighting for Go code.
* Never asks before saving or quitting. Be careful.
* Random characters may appear on the screen when keys are pressed. Clear them with `ctrl-l`.
* `Home` and `End` are not detected by the key handler. VT100 only.
* Can format Go code using `gofmt` (press `ctrl-f`).
* Expects utilities like `gofmt` to be in `/usr/bin`.
* Will strip trailing whitespace whenever it can.
* Must be given a filename at start.
* Requires that `/dev/tty` is available.
* Copy, cut and paste are only for one line at a time.
* There may be issues when resizing the terminal.
* Some letters can not be typed. Like `ø`.
* If a line contains a unicode character (like `ø`), the cursor may be misplaced after that position.

## Hotkeys

* `ctrl-q` to quit
* `ctrl-s` to save
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
* `ctrl-x' to cut the current line
* `ctrl-c' to copy the current line
* `ctrl-v' to paste the current line
* `ctrl-b` to bookmark the current position
* `ctrl-j` to jump to the bookmark
* `ctrl-h` to show a minimal help text
* `ctrl-u` to undo
* `ctrl-z` to undo, but this may also background the editor ("fg" to foreground)
* `esc` to toggle syntax highlighting

## Size

The `o` executable is **444k** when built with GCC 9.1 (for 64-bit Linux):

    go build -gccgoflags '-Os -s'

For comparison, it's **2.8M** when building with Go 1.13 and no particular build flags are given.

## General info

* Version: 2.1.0
* License: MIT
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
