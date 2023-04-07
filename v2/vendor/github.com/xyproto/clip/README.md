[![GoDoc](https://godoc.org/github.com/xyproto/clip?status.svg)](http://godoc.org/github.com/xyproto/clip)

# Clipboard for Go

This is a fork of [clipboard](https://github.com/atotto/clipboard) (which is licensed under the "BSD 3-clause" license).

The goal is to provide functionality for copying from and pasting to the clipboard, while also merging in most pull requests that `atotto/clipboard` has received.

### Requirements

* Go 1.13 or later

### Build

    $ go get -u github.com/xyproto/clip

### Platforms

* macOS
* Windows 7 (probably works on later editions too)
* Linux, Unix (requires `xclip` and `xsel` to be installed (or `wlpaste` and `wlcopy`, for Wayland)

### Online documentation

* http://godoc.org/github.com/xyproto/clip

### Notes

* For functions that takes or return a string, only UTF-8 is supported.

## Commands

Paste shell command:

    $ go get -u github.com/xyproto/clip/cmd/gopaste
    $ # example:
    $ gopaste > document.txt

Copy shell command:

    $ go get -u github.com/xyproto/clip/cmd/gocopy
    $ # example:
    $ cat document.txt | gocopy

### General info

* Version: 0.3.2
* License: BSD-3
