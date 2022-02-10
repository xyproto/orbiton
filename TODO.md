# TODO

## Building and testing programs

- [ ] Jump to error when building with `ctrl-space` and `cargo`.
- [ ] Fix output parsing when running `go test` with ctrl-space.

## Code navigation

- [ ] When pressing ctrl-g or F12 and there's a filename under the cursor that exists, go to that file.
- [ ] When pressing ctrl-g on a function that is declared in a file in the same directory, go to that file and function definition.
- [ ] `ctrl-f` and then `return` could jump to a location at least 10 lines away that has been most visited within the last 10 minutes.

## Code editing

- [ ] When commenting out a block, move comment markers closer to the beginning of the text.
- [ ] When sorting comma-separated strings that do not start with (, [ or {, make sure to keep the same trailing comma status.
- [ ] When `}` is the last character of a file, sometimes pressing enter right before it does not work.
- [ ] If there are four lines: not comment, comment, not comment, comment, let ctrl+/ behave differently.
- [ ] When moving far to the right of a long line, `ctrl-k` sometimes cuts from the wrong place.
- [ ] Indentation in Rust is sometimes wonky.
- [ ] Fix the ctrl-\ behavior when commenting out a block at the end of a file.
- [ ] When changing a file from tabs to spaces, or the other way around, also modify indentations after comment markers.
- [ ] Tab in the middle of a line, especially on a `|` character, could insert spaces until the `|` aligns with the `|` above,
      if applicable (For Markdown tables).
- [ ] Smarter indentation for `}`. There are still a few cases where it's not too smart.
      Perhaps use the logic for tab-indenting for when dedenting `}`?
- [ ] If joining a line that starts with a single-line comment with a line below that also starts with a single line comment,
      remove the extra comment marker.
- [ ] When in "SuggestMode", typing should start filtering the list.

## Debugging

- [ ] Let `ctrl-space` compile C and C++ programs with debug flags if debug mode is enabled.
- [ ] Be able to run a C program until the breakpoint and then enter gdb with the source file loaded.
- [ ] Support Delve. Introduce a Debug mode.

## Autocompletion

- [ ] Let the auto-completion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Autocompletion of filenames if the previous rune is "/" and tab is pressed.

## Saving and loading

- [ ] When editing a man page, make it possible to toggle between the man page and the man page view mode.
- [ ] Auto-detect tabs/spaces when opening a file.
- [ ] When editing a file that then is deleted, `ctrl-s` should maybe create the file again?
      Or save it to `/tmp` or `~/.cache/o`? Or copy it to the clipboard?
- [ ] Be able to open and edit large text files (60M+).
- [ ] Should be able to open any binary file and save it again, without replacements. Add a hex edit mode.
- [ ] Auto-detect if a loaded file uses `\t` or 1, 2, 3, 4, or 8 spaces for indentation.
- [ ] Introduce a hexedit mode that will:
      * Not load the entire file into memory.
      * Display all bytes as a grid of "0xff" style fields, with the string representation to the right.
      * This might be better solved by having a separate hex editor?
- [ ] Be able to edit `.txt.gz` and `.1.gz` files.
- [ ] Plugins. When there's `txt2something` and `something2txt`, o should be able to edit "something" files in general.
      This could be used for hex editing, editing ELF files etc.
- [ ] When the editor executable is `list`, just list the contents and exit?

## Syntax highlighting

- [ ] Check that the right theme is loaded under `uxterm`.
- [ ] Don't let single-line comments at the end of lines disable rainbow parentheses.
- [ ] Don't highlight /* in shell scripts like comments.
- [ ] Also highlight hexadecimal numbers.
- [ ] Fix syntax highlighting of `'tokens` in Clojure.
- [ ] '://' should not be interpreted as starting a single-line comment. It's very likely to be an URL.
- [ ] Don't highlight regular text in Nroff files.
- [ ] `-- ` comments in Ada are not recognized.
- [ ] Links in Markdown documents are not always recognized.
- [ ] // within a ` block should not be recognized
- [ ] Opening a read-only file in the Linux terminal displays different red colors when moving to the bottom.
- [ ] Highlight this shell script line correctly: `for txt in third_party/*.txt; do`
- [ ] Don't color `#` as a comment when editing Go code.
- [ ] If a word over N letters is typed 1 letter differently from all the other instances in the current file: color it differently!
- [ ] Rainbow parenthesis should be able to span multiple lines, especially for Clojure, Common Lisp, Scheme and Emacs Lisp.
- [ ] Hash strings (like sha256 hash sums), could be colored light yellow and dark yellow for every 2 characters
- [ ] When viewing man pages, stop the file from being possible to edit, but use the default color theme.
- [ ] Spellcheck all comments that are in English. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] Shell scripts with if/else/endif blocks that are commented out are highlighted wrong.
- [ ] Ignore multiline comments within multiline comments.
- [ ] Also enable rainbow parenthesis for lines that ends with a single-line comment.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Highlight links in Markdown (perhaps color `[` and `]` yellow).

## Documentation

- [ ] Replace ` in o.1 with \b.

## Cut, copy, paste and portals

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
- [ ] Don't complain about the mid-dot or other unicode symbols that can be displayed in the space of 1 character. Use the unicode width module.
- [ ] Better support for multi-byte unicode runes.
- [ ] Open text files with Chinese/Japanese/Korean characters without breaking the text flow.
- [ ] Quotestate Process can not recognize triple runes, like the previous previous rune is ", the previous rune is " and the current rune is ".
      The wrong arguments are passed to the function. Figure out why.
- [ ] If over a certain percentage of the characters are not unicode.Graphics, enter binary mode.

## Command menu

- [ ] Command menu option for deleting the rest of the file.
- [ ] A menu option for recording simple vim-style keypresses, then a keypress for playing it back.
- [ ] Add word wrap with a custom line length to the command menu.
- [ ] Insert a custom file from the command menu.
- [ ] Add one or more of these commands: regex search, go to definition, rename symbol, find references and disassembly.
- [ ] Make it easy to make recordings of the editing process.

## Localization

- [ ] Localize all status messages and menu options.

## External programs

- [ ] Let `guessica` also set `pkgrel=1` if there was a new version.
- [ ] Embed fstabfmt.
- [ ] Extract the functionality for searching a MessagePack file to a `mpgrep` utility, that has a `-B` flag (like `grep`).
- [ ] Draw inspiration from [kilo](https://github.com/antirez/kilo).

## Unit tests

- [ ] Add tests for the smart indent feature: for pressing return, tab and space, especially in relation with `{` and `}`.

## Performance

- [ ] Skip `os.Stat` and `Glob` if they take to long, and just open the file directly (they are needed for smart filename completion).
- [ ] Extract the features that are used in `vt100` and create a more optimized package.
- [ ] Reduce memory usage.

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
- [ ] Abstract the editor, so that sending in keypresses and examining the result can be tested.
- [ ] Rewrite `insertRune`. Improve word-wrap related functionality.
- [ ] Introduce a type for screen coordinates, a type for screen coordinates + scroll offset, and another type for data coordinates.
