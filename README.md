# o [![Build Status](https://travis-ci.com/xyproto/o.svg?branch=master)](https://travis-ci.com/xyproto/o) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/o)](https://goreportcard.com/report/github.com/xyproto/o) [![License](https://img.shields.io/badge/license-BSD-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/o/master/LICENSE)

`o` is a small and fast text editor that is limited to the VT100 standard.

It's a good fit for writing git commit messages, using `EDITOR=o git commit`.

For a more feature complete editor that is also written in Go, check out [micro](https://github.com/zyedidia/micro).

## Packaging status

[![Packaging status](https://repology.org/badge/vertical-allrepos/o-editor.svg)](https://repology.org/project/o-editor/versions) [![Packaging status](https://repology.org/badge/vertical-allrepos/o.svg)](https://repology.org/project/o/versions)

## Quick start

You can install `o` with Go 1.10 or later:

    go get -u github.com/xyproto/o

## Setting `o` as the default editor for `git`

To set:

    git config --global core.editor o

To unset:

    git config --global --unset core.editor

## Features and limitations

* Loads up instantly.
* Small executable size (around 500k, when built with `gccgo` and then stripped).
* Provides syntax highlighting for Go, C++, Markdown and Bash. Other files may also be highlighted (toggle with `ctrl-t`).
* Configuration-free, for better and for worse.
* Is limited to the VT100 standard, so hotkeys like `ctrl-a` and `ctrl-e` must be used instead of `Home` and `End`.
* Compiles with either `go` or `gccgo`.
* Tested with `st`, `urxvt` and `xfce4-terminal`.
* Tested on Arch Linux and FreeBSD.
* Loads faster than both `vim` and `emacs`.
* Never asks before saving or quitting. Be careful!
* Can format Go or C++ code, just press `ctrl-space`. This uses either `goimports` (`go get golang.org/x/tools/cmd/goimports`) or `clang-format`.
* Will strip trailing whitespace whenever it can.
* Will replace nonbreaking space (`0xc2 0xa0`) with a regular space (`0x20`) whenever it can.
* Must be given a filename at start.
* Can provide smart indentation.
* Requires `/dev/tty` to be available.
* Copy, cut and paste is only for one line at a time. `xclip` (for X) or `wl-clipboard` (for Wayland) must be installed if the system clipboard is to be used.
* Only forward search.
* May take a line number as the second argument, with an optional `+` prefix.
* The text will be red if a loaded file is read-only.
* The terminal needs to be resized to show the second half of lines that are longer than the terminal width.
* If the filename is `COMMIT_EDITMSG`, the look and feel will be adjusted for git commit messages.
* There's currently no good way to move blocks of code around, but this will be added.
* Supports `UTF-8`.
* Respects the `NO_COLOR` environment variable.
* Can render text to PDF.
* Only UNIX-style line endings are supported (`\n`).
* Will convert DOS/Windows line endings (`\r\n`) to UNIX line endings (just `\n`), whenever possible.
* Will jump to the last visited line when opening a recent file.
* If tab completion on the terminal went wrong and you are trying to open a `main.` file that does not exist, but `main.cpp` and `main.o` does exists, then `main.cpp` will be opened.

## Known bugs

* Files with lines longer than the terminal width are not supported (resize the terminal to edit long lines).
* When elongating a long line (one that is already longer than the terminal width), the line will break at an unexpected location.

## Hotkeys

* `ctrl-q` - Quit
* `ctrl-s` - Save
* `ctrl-w` - Format the current file using `goimport` or `clang-format`, depending on the file extension.
* `ctrl-a` - Go to start of text, then start of line and then to the previous line.
* `ctrl-e` - Go to end of line and then to the next line.
* `ctrl-p` - Scroll up 10 lines.
* `ctrl-n` - Scroll down 10 lines, or go to the next match if a search is active.
* `ctrl-k` - Delete characters to the end of the line, then delete the line.
* `ctrl-g` - Toggle a status line at the bottom for displaying: filename, line, column, Unicode number and word count.
* `ctrl-d` - Delete a single character.
* `ctrl-t` - Toggle syntax highlighting.
* `ctrl-o` - Toggle between text and draw mode.
* `ctrl-x` - Cut the current line.
* `ctrl-c` - Copy the current line.
* `ctrl-v` - Paste the current line.
* `ctrl-b` - Bookmark the current line.
* `ctrl-j` - Jump to the bookmark (or join lines if a bookmark is not set).
* `ctrl-u` - Undo (`ctrl-z` is also possible, but may background the application).
* `ctrl-l` - Jump to a specific line number.
* `ctrl-f` - Search for a string.
* `esc` - Redraw the screen and clear the last search.
* `ctrl-space` - Build Go or C++ files, word-wrap other files.
* `ctrl-\` - Toggle single-line comments for a block of code.
* `ctrl-r` - Render the current text as a PDF document.
* `ctrl-~` - Save and quit.

Additionally:

* If interactive rebase is launched with `git rebase -i`, either `ctrl-r` or `ctrl-w` will cycle the keywords for the current line (`fixup`, `drop`, `edit` etc).
* When editing Markdown, checkboxes can be toggled with `ctrl-w`.

## Manual installation

On Linux:

    git clone https://github.com/xyproto/o
    cd o
    go build -mod=vendor
    sudo install -Dm755 o /usr/bin/o
    gzip o.1
    sudo install -Dm644 o.1.gz /usr/share/man/man1/o.1.gz

## Dependencies

C++

* For building code with `ctrl-space`, [`cxx`](https://github.com/xyproto/cxx) must be installed.
* For formatting code with `ctrl-w`, `clang-format` must be installed.

Go

* For building code with `ctrl-space`, The `go` compiler must be installed.
* For formatting code with `ctrl-w`, [`goimports`](https://godoc.org/golang.org/x/tools/cmd/goimports) must be installed.

Zig

* For building and formatting Zig code, only the `zig` command is needed.

V

* For building and formatting V code, only the `v` command is needed.

Rust

* For building code with `ctrl-space`, `Cargo.toml` must exist and `cargo` must be installed.
* For formatting code with `ctrl-w`, `rustfmt` must be installed.

## Size

* The `o` executable is only **508k** when built with GCC 9.3 (for 64-bit Linux) + compressed with `upx`.
* This isn't as small as [e3](https://sites.google.com/site/e3editor/), an editor written in assembly (which is **234k**), but it's reasonably lean.

One way of building with `gccgo` and `upx`:

    go build -mod=vendor -gccgoflags '-Os -s' && upx o

It's **5.2M** when built with Go 1.14 and no particular build flags are given.

## Jumping to a specific line when opening a file

These four ways of opening `file.txt` at line `7` are supported:

* `o file.txt 7`
* `o file.txt +7`
* `o file.txt:7`
* `o file.txt+7`

This also means that filenames containing `+` or `:`, and then followed by a number, are not supported.

## The first keypress

If the very first keypress after starting is `G` or `/`, it will trigger the following vi-compatible behavior:

* `/` - enter search-mode (same as when pressing `ctrl-f`)
* `G` - go to the end of the file
* `O` - if the following key is uppercase and not 'K' (because of "OK")

The reason for adding these is to make using `o` easier to use for long-time vi/vim/neovim users.

## Spinner

When loading large files, an animated spinner will appear. The loading operation can be interrupted by pressing `esc`, `q` or `ctrl-q`.

![progress](img/progress.gif)

## Easter eggs

Pressing `ctrl-space` will render Markdown files to PDF using `pandoc` (as opposed to `ctrl-r`, which will save the text directly to a PDF, without using `pandoc`).

## Shell function

This shell function works in `zsh` and `bash` and may be useful for both searching for and opening a file at the given line number (works best if there is only one matching file):

```bash
fo() { find . -type f -wholename "*$1" -exec /usr/bin/o {} $2 \;; }
```

Example use:

```sh
fo somefile.cpp 123
```

## General info

* Version: 2.24.2
* License: 3-clause BSD
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
