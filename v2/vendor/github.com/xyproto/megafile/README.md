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

**Navigation and Selection**
* `↑/↓/←/→` - navigate and select files
* `Page Up/Down` - jump to first/last entry in current column
* `Home` or `ctrl-a` - jump to first file (or start of line when typing)
* `End` or `ctrl-e` - jump to last file (or end of line when typing)

**Execution**
* `Return` - execute selected file, or run typed command
* `Esc` - clear selection (first press), exit program (second press)

**Text Editing**
* `Backspace` - delete character, or go up directory (when at start)
* `ctrl-h` - delete character, or toggle hidden files (when at start)
* `ctrl-d` - delete character under cursor, or exit program
* `ctrl-k` - delete text to the end of the line
* `ctrl-c` - clear text, or exit program

**File Operations**
* `Tab` - cycle through files, or tab completion
* `Delete` - move selected file to trash (when no text is typed)
* `ctrl-z` or `ctrl-u` - undo last trash move
* `ctrl-f` - search for text in files

**Directory Navigation**
* `ctrl-space` - enter the most recent subdirectory
* `ctrl-n` - cycle to next directory
* `ctrl-p` - cycle to previous directory
* `ctrl-b` - go to parent directory

**Display**
* `ctrl-o` - toggle show hidden files
* `ctrl-l` - clear screen

**External Tools**
* `ctrl-t` - run `tig`
* `ctrl-g` - run `lazygit`

**Exit**
* `ctrl-q` - exit program immediately

### Runtime dependencies

* `tig`
* `lazygit`
* `/bin/sh` for displaying the uptime on macOS

### General info

* Version: 1.4.0
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
