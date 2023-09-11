# TODO

## General

- [ ] `ctrl-g`, `up` could go to the previous function signature.
- [ ] `ctrl-g`, `down` could go to the next function signature.
- [ ] Let the `ctrl-g` status line also contain the line percentage, like `ctrl-c` in Nano.
- [ ] Save a "custom words" and "ignored words" list to disk.
- [ ] Also timestamp the search history so that it can be cropped when it grows too long.
- [ ] If in man page mode, set the file as read-only and also let `q` quit.
- [ ] Let `ctrl-w` also format gzipped code, for instance when editing `main.cpp.gz`.
- [ ] Do not remove indentation from JS code in HTML when `ctrl-w` is pressed. See: https://github.com/yosssi/gohtml/issues/22
- [ ] When rebasing, look for the `>>>>` markers when opening the file and jump to the first one (and let `ctrl-n` search for the next one).
- [ ] When pasting with _double_ `ctrl-v`, let _one_ `ctrl-z` undo both keypresses.
- [ ] When pasting lines that start with `+` and it's not a diff/patch file, then replace `+` with a blank.
- [ ] When deleting lines with `ctrl-k` more than once, scroll the cursor line a bit up, to make it easier.
- [ ] If a file is passed through stdin and > 70% of the lines has a `:`, it might be a log file and not configuration.
- [ ] If a file is passed through stdin and has many similar lines and no comments or blank lines, it might be a log file and not configuration.
- [ ] HTTP client - scratch document style `.http` files.
- [ ] Add support for emojis. Perhaps by drawing lines differently if an emoji is detected.
- [ ] Parse some programming languages to improve the quality of the "go to defintion" feature.
- [ ] When the last line in a document is a long line ending with "}", make it possible to press return before the "}".
- [ ] Make it possible to export code to HTML or PNG, maybe by using Splash.
- [ ] Go through this file and remove all completed TODO items.

### Nano emulation mode

- [ ] If the file is huge, let ctrl-t time out instead of waiting for it to complete.
- [ ] When searching for a typo with ctrl-t, enable wrap-around for the search.
- [ ] Make the spell check dictionary persitent.
- [ ] Make it possible to set a marker with a hotkey before pressing ctrl-k.
- [ ] ctrl-\\ for replace (use ctrl-w, type in text to search for and then press Tab instead of Return).
- [ ] ctrl-q for searching backwards.
- [ ] alt-e for redo.
- [ ] alt-6 for copy (use ctrl-c instead).
- [ ] alt-w for next (use ctrl-n instead).
- [ ] alt-u for undo (use ctrl-z instead, if possible).
- [ ] alt-q for previous (use ctrl-p instead).
- [ ] alt-a to set mark.
- [ ] alt-] to jump to bracket.
- [ ] Support other themes, like the Mono Gray theme.
- [ ] Add ctrl-a to add a word to the dictionary after searching with ctrl-t to the help overview.

See also: https://staffwww.fullcoll.edu/sedwards/nano/nanokeyboardcommands.html

## `o` to GUI frontend communication

- [ ] When changing themes from within the VTE/GKT3 frontend, let `o` be able to communicate a palette change per theme, using some sort of RPC.
- [ ] Use proper RPC between `o` and the VTE/GTK3 frontend. This also helps when upgrading to GTK4.
- [ ] Create an SDL2 frontend.

## Maybe

- [ ] Highlight changed lines if a file changed while monitoring it with `-m`.
- [ ] Move redrawing and clearing the statusbar to a separate goroutine.

## Markdown

- [ ] `ctrl-space` is too easy to press by accident, find a better solution.

## Autocompletion and AI generated code

- [ ] If ChatGPT is enabled, and there is just one error, and the fix proposed by ChatGPT is small, then apply the fix, but let the user press `ctrl-z` if they don't want it.
- [ ] If an API key is entered, save it to file in the cache directory.
- [ ] Add an environment variable for specifying the AI API endpoint.
- [ ] Embed https://github.com/nomic-ai/gpt4all + data files within the `o` executable, somehow.
- [ ] Add a way to generate git commit messages with ChatGPT
- [ ] Let the auto completion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Auto completion of filenames if the previous rune is `/` and tab is pressed.
- [ ] When generating code with ChatGPT, also send a list of function signatures and constants for the current file (+ header file).

## Building, debugging and testing programs

- [ ] Along with the per-file location, store the per-file last `ctrl-o` menu choice location. Or just move "Build" to the top, when on macOS.
- [ ] Jump to error for Erlang.
- [ ] Fix output parsing when running `go test` with `ctrl-space`.
- [ ] Jump to error when building with `ctrl-space` and `cargo`.
- [ ] When switching register pane layout with `ctrl-p`, save the contents of the old pane and use that.
- [ ] Make it possible to send custom commands to `gdb` with `ctrl-g` when in debug mode.
- [ ] Make it possible to step through Go programs as well.
- [ ] Build Jakt and Prolog programs with ctrl-space.
- [ ] Support for Prolog.

## Saving and loading

- [ ] When "somefile.go" and "somefile_test.go" exists, and only "somefile" is given, load "somefile.go".
- [ ] When a filename is given, but it does not exist, and no extension is given, and the directory only contains one file, open that one.
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

- [ ] Make it possible to have groups of bookmarks per file, and then name them, somehow.
- [ ] When pressing `ctrl-g` or `F12` and there's a filename under the cursor that exists, go to that file.
- [ ] When pressing `ctrl-g` on a function that is declared in a file in the same directory, go to that file and function definition.
- [ ] `ctrl-f` and then `return` could jump to a location at least 10 lines away that has been most visited within the last 10 minutes.
- [ ] `ctrl-f` twice should search for the current word.
- [ ] Let `ctrl-l` double as a command prompt.
- [ ] When bookmarking, don't just bookmark the line/col, but also the filename. Maybe.

## Code editing

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
- [ ] When in `SuggestMode`, typing should start filtering the list.
- [ ] Introduce the concept of soft and hard breaks, to keep track of where lines were broken automatically and be able to reflow the text.
- [ ] Sort lines in a less opaque and unusual way than `left,up,right` `sort` `return` before documenting the feature.
- [ ] Let ctrl-k first delete until "{" and then util the end of the line if there is no "{"?

## Syntax highlighting

- [ ] Fix syntax highlighting of `(* ... *)` comments at the end of a line in OCaml.
- [ ] When viewing man pages, respect the current theme.
- [ ] Let `<<EOF` be considered the start of a multiline string in Shell, and `EOF` the end.
- [ ] Let a struct for a Theme contain both the light and the dark version, if there are two.
- [ ] Check that the right theme is loaded under `uxterm`.
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

- [ ] Support pbcopy/pbpaste (`echo asdf | pbcopy -Prefer txt` and `echo $(pbpaste -Prefer txt)`).
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
- [ ] Quotestate Process can not recognize triple runes, like the previous previous rune is ", the previous rune is " and the current rune is ". The wrong arguments are passed to the function. Figure out why.

## Command menu

- [ ] Add a menu option for listing all functions in the current directory, alphabetically, and be able to jump to any one of them.
- [ ] Add one or more of these commands: regex search, go to definition, rename symbol, find references and disassembly.
- [ ] Make it easy to make recordings of the editing process (can already use asciinema?).

## Localization

- [ ] Localize all status messages and menu options.

## External programs

- [ ] Let rendering with `pandoc` have a spinner, since it can take a little while.
- [ ] Extract the functionality for searching a MessagePack file to a `mpgrep` utility, that has a `-B` flag (like `grep`).
- [ ] Draw inspiration from [kilo](https://github.com/antirez/kilo).

## Unit tests

- [ ] Add tests for the smart indent feature: for pressing return, tab and space, especially in relation with `{` and `}`.

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
- [ ] Consider switching over to `github.com/creack/pty`, for better multi-platform support.

## Built-in game

- [ ] Two pellets next to each other should combine.
