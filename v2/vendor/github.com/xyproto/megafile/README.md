# MegaFile

A simple and colorful TUI shell for Linux, written in Go.

This project can be used as the `github.com/xyproto/megafile` package, or as a standalone exectuable.

![screenshot](img/screenshot.png)

(screenshot of the previous version, before the project rename)

### Building

    make

### Installation

    make install

### Commands

* any filename - edit file with `$EDITOR`
* `cd`, `..` or any directory name - change directory
* `./script.sh` - execute a script named `script.sh`
* `ls` or `dir` list directory (happens automatically, though)
* `q`, `quit` or `exit` - exit program

### Hotkeys

* `tab` - cycle between the 3 different current diretories, or tab completion of directories and filenames
* `ctrl-space` - cycle backwards between the 3 different current directories
* `ctrl-q` - exit program
* `ctrl-o` or `ctrl-h` - toggle "show hidden files"
* `ctrl-a` - start of line
* `ctrl-d` - delete character under cursor, or exit program
* `ctrl-k` - delete text to the end of the line
* `ctrl-l` - clear screen
* `ctrl-c` - clear text, or exit program
* `ctrl-t` - run `tig`
* `ctrl-g` - run `lazygit`
* `ctrl-n` - enter the freshest directory
* `ctrl-p` - go up one directory

### General info

* Version: 1.2.1
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
