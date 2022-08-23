# TODO

## General

- [ ] Go through this file and remove all completed TODO items.

## Building, debugging and testing programs

- [ ] Jump to error for Erlang.
- [ ] Fix output parsing when running `go test` with ctrl-space.
- [ ] Jump to error when building with `ctrl-space` and `cargo`.
- [ ] When switching register pane layout with `ctrl-p`, save the contents of the old pane and use that.
- [ ] Make it possible to send custom commands to `gdb` with `ctrl-g` when in debug mode.
- [ ] Make it possible to step through Go programs as well.
- [ ] Build Jakt and Prolog programs with ctrl-space.
- [ ] Support for Prolog.

## Saving and loading

- [ ] When "somefile.go" and "somefile_test.go" exists, and only "somefile" is given, load "somefile.go".
- [ ] Show a spinner when reading a lot of data from stdin.
- [ ] When editing a man page, make it possible to toggle between the man page and the man page view mode.
- [ ] Auto-detect tabs/spaces when opening a file.
- [ ] When editing a file that then is deleted, `ctrl-s` should maybe create the file again?
      Or save it to `/tmp` or `~/.cache/o`? Or copy it to the clipboard?
- [ ] Be able to open and edit large text files (60M+).
- [ ] Auto-detect if a loaded file uses `\t` or 1, 2, 3, 4, or 8 spaces for indentation.
- [ ] Introduce a hexedit mode for binary files that will:
      * Not load the entire file into memory.
      * Display all bytes as a grid of "0xff" style fields, with the string representation to the right.
      * This might be better solved by having a separate hex editor?
- [ ] Be able to edit `.txt.gz` and `.1.gz` files.
- [ ] Plugins. When there's `txt2something` and `something2txt`, o should be able to edit "something" files in general.
      This could be used for hex editing, editing ELF files etc.
- [ ] When the editor executable is `list`, just list the contents and exit?

## Code navigation

- [ ] When pressing `ctrl-g` or `F12` and there's a filename under the cursor that exists, go to that file.
- [ ] When pressing `ctrl-g` on a function that is declared in a file in the same directory, go to that file and function definition.
- [ ] `ctrl-f` and then `return` could jump to a location at least 10 lines away that has been most visited within the last 10 minutes.

## Code editing

- [ ] When moving far to the right of a long line, `ctrl-k` sometimes cuts from the wrong place.
- [ ] When commenting out a block, move comment markers closer to the beginning of the text.
- [ ] When sorting comma-separated strings that do not start with (, [ or {, make sure to keep the same trailing comma status.
- [ ] When `}` is the last character of a file, sometimes pressing enter right before it does not work.
- [ ] If there are four lines: not comment, comment, not comment, comment, let ctrl+/ behave differently.
- [ ] Indentation in Rust is sometimes wonky.
- [ ] When changing a file from tabs to spaces, or the other way around, also modify indentations after comment markers.
- [ ] Tab in the middle of a line, especially on a `|` character, could insert spaces until the `|` aligns with the `|` above,
      if applicable (For Markdown tables).
- [ ] Smarter indentation for `}`. There are still a few cases where it's not too smart.
      Perhaps use the logic for tab-indenting for when dedenting `}`?
- [ ] If joining a line that starts with a single-line comment with a line below that also starts with a single line comment,
      remove the extra comment marker.
- [ ] When in "SuggestMode", typing should start filtering the list.
- [ ] Introduce the concept of soft and hard breaks, to keep track of where lines were broken automatically and be able to reflow the text.

## Autocompletion

- [ ] Let the auto completion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Auto completion of filenames if the previous rune is "/" and tab is pressed.

## Syntax highlighting

- [ ] Let `<<EOF` be considered the start of a multiline string in Shell, and `EOF` the end.
- [x] Let a struct for a Theme contain both the light and the dark version, if there are two.
- [ ] Check that the right theme is loaded under `uxterm`.
- [ ] Don't let single-line comments at the end of lines disable rainbow parentheses.
- [ ] Also highlight hexadecimal numbers.
- [ ] Fix syntax highlighting of `'tokens` in Clojure.
- [ ] Don't highlight regular text in Nroff files.
- [ ] `-- ` comments in Ada should be recognized.
- [ ] // within a ` block should not be recognized
- [ ] Opening a read-only file in the Linux terminal should not display different red colors when moving to the bottom.
- [ ] If a word over N letters is typed 1 letter differently from all the other instances in the current file: color it differently!
- [ ] Rainbow parenthesis should be able to span multiple lines, especially for Clojure, Common Lisp, Scheme and Emacs Lisp.
- [ ] Hash strings (like sha256 hash sums), could be colored light yellow and dark yellow for every 2 characters
- [ ] Spellcheck all comments that are in English. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] Ignore multiline comments within multiline comments.
- [ ] Also enable rainbow parenthesis for lines that ends with a single-line comment.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Highlight links in Markdown (perhaps color `[` and `]` yellow).

## Documentation

- [ ] Replace ` in o.1 with \b.
- [ ] Document that pressing the arrow keys in rapid succession and typing in `!sort` can sort a block of text with the external `sort` command.

## Cut, copy, paste and portals

- [ ] Make it possible to double press `ctrl-c` again, to also copy the next block of text.
- [ ] Let `ctrl-t` take a line and move it through the portal?
- [ ] GUI: Look into the clipboard functions for VTE and if they can be used for mouse copy + paste.
- [ ] Re-enable cross-user portals?
- [ ] Paste with middle mouse button.
- [ ] When starting o, hash sum the clipboards it can find. When pasting, use the latest changed clipboard. If nothing changed, use the one for Wayland or X11, depending on environment variables.
- [ ] Use `wl-copy` for copy and cut. Use the same type of implementation as for `wl-paste`.
- [ ] Figure out why copy/paste is wonky on Wayland.
- [ ] Check if Wayland is in use (env var) before using `wl-*`.
- [ ] Add a command menu option to copy the entire file to the clipboard.
- [ ] Add a command menu option to copy the build command to the clipboard.
- [ ] If `xclip` or similar tool is not available, cut/copy/paste via a file.
- [ ] Pressing `ctrl-v` to paste does not work across X/Wayland sessions. It would be nice to find a more general clipboard solution.
- [ ] Let the cut/copy/paste line state be part of the editor state, because of undo.
- [ ] Cross user portals? Possibly by using `TMPDIR/oportal.dat`.

## Encoding

- [ ] Detect ISO-8859-1 and convert the file to UTF-8 before opening.
- [ ] Open text files with Chinese/Japanese/Korean characters without breaking the text flow.
- [ ] Quotestate Process can not recognize triple runes, like the previous previous rune is ", the previous rune is " and the current rune is ".
      The wrong arguments are passed to the function. Figure out why.

## Command menu

- [ ] Add one or more of these commands: regex search, go to definition, rename symbol, find references and disassembly.
- [ ] Make it easy to make recordings of the editing process (can already use asciinema?).

## Localization

- [ ] Localize all status messages and menu options.

## External programs

- [ ] Let rendering with `pandoc` have a spinner, since it can take a little while.
- [ ] Let `guessica` also set `pkgrel=1` if there was a new version.
- [ ] Embed `fstabfmt`.
- [ ] Extract the functionality for searching a MessagePack file to a `mpgrep` utility, that has a `-B` flag (like `grep`).
- [ ] Draw inspiration from [kilo](https://github.com/antirez/kilo).

## Unit tests

- [ ] Add tests for the smart indent feature: for pressing return, tab and space, especially in relation with `{` and `}`.

## Performance

- [ ] Skip `os.Stat` and `Glob` if they take to long, and just open the file directly (they are needed for smart filename completion).
- [ ] Extract the features that are used in `vt100` and create a more optimized package.
- [ ] Reduce memory usage even further.

## Refactoring

- [ ] Replace and/or rewrite the syntax highlighting system.
- [ ] Refactor the code to handle a line as a Line struct/object that has these markers: start of line, start of text, start of scroll view,
      end of scroll view, end of text, one after end of text, end of line including whitespace.
- [ ] Inherit from the Line struct (with interfaces+types+methods) by adding per-language markers: start of block, end of block,
      indentation compared to the line above, dedentation compared to the line above
- [ ] Or go for a server/client type of model, where the server deals with moving around in very large files, for instance.
- [ ] Create a Go package for detecting:
      * Language (specifically C++98 for instance)
      * Language family (C-like, ML-like etc)
      * tabs, spaces, indentations, mixed tabs/spaces
      * clang style, so that the same style may be used?
      * emacs tag
      * vim tag
      * what else?
      * then translate this to a struct
      * also think about how this can be skipped is the file is enormous and should be read in block-by-block
- [ ] Abstract the editor, so that sending in keypresses and examining the result can be tested with Go tests.
- [ ] Rewrite `insertRune`. Improve word-wrap related functionality.
- [ ] Introduce a type for screen coordinates, a type for screen coordinates + scroll offset, and another type for data coordinates.
- [ ] Create a Terminal type that implement the context.Context interface, then pass that to functions that
      would otherwise take both a `vt100.Canvas`, `vt100.TTY` and a `StatusBar`.

## Built-in game

- [ ] Two pellets next to each other should combine.
