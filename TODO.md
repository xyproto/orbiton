# TODO

## Pri

- [ ] Paste with middle mouse button.
- [ ] Abstract the editor, so that sending in keypresses and examining the result can be tested.
- [ ] Better support for multi-byte unicode runes.

## Bugs/features/issues

- [ ] Jump to error when building with `ctrl-space` and `cargo`.
- [ ] Fix the `editor.Delete()` method, so that `ctrl-d` also works for very long lines.
- [ ] Be able to open and edit large text files (60M+).
- [ ] Auto-detect tabs/spaces when opening a file.
- [ ] Open text files with Chinese/Japanese/Korean characters without breaking the text flow.
- [ ] `ctrl-j` does not seem to do anything if pressed at the end of a long line.
- [ ] When pressing `ctrl-space` the first time, have a timeout before registering as a double press.
- [ ] Hash strings (like sha256 hash sums), could be colored light yellow and dark yellow for every 2 characters
- [ ] When the editor executable is `list`, just list the contents and exit?
- [ ] When running the `red` editor, don't draw the other theme first.
- [ ] Don't redraw everything when just moving the cursor around.
- [ ] Don't color `#` as a comment when editing Go code.
- [ ] Use `wl-copy` for copy and cut. Use the same type of implementation as for `wl-paste`.
- [ ] Figure out why copy/paste is wonky on Wayland.
- [ ] Figure out why a file saved by another user can not be read by Go in `/tmp`, but can be read with `cat`.
- [ ] Command menu option for deleting the rest of the file.
- [ ] A menu option for recording simple vim-style keypresses, then a keypress for playing it back.
- [ ] Highlight the line where a portal has been opened.
- [ ] Highlight the part of git commit topics that are >= 80 characters long.
- [ ] For git commit text, highlight column 80 if the text crosses that boundry.
- [ ] Check if Wayland is in use before using `wl-*`.
- [ ] Add a command menu option to copy the entire file to the clipboard.
- [ ] Add a command menu option to copy the build command to the clipboard.
- [ ] If `xclip` or similar tool is not available, cut/copy/paste via a file.
- [ ] Pressing `ctrl-v` to paste does not work across X/Wayland sessions. It would be nice to find a more general clipboard solution.
- [ ] Let the cut/copy/paste line state be part of the editor state, because of undo.
- [ ] Insert `# vim: ts=2 sw=2 et:` at the bottom when `ctrl-space` is pressed in a PKGBUILD file. Or add a command menu option for this.
- [ ] Autocompletion of filenames if the previous rune is "/" and tab is pressed.
- [ ] Add word wrap with a custom line length to the command menu.
- [ ] If a word over N letters is typed 1 letter differently from all the other instances in the current file: color it differently.
- [ ] Spellcheck all comments that are in English. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] Should be able to open any binary file and save it again, without replacements. Add a hex edit mode.
- [ ] Detect ISO-8859-1 and convert the file to UTF-8 before opening.
- [ ] Auto-detect if a loaded file uses `\t` or 1, 2, 3, 4, or 8 spaces for indentation.
- [ ] Let the autocompletion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Cross user portals, possibly by using `TMPDIR/oportal.dat`.
- [ ] Add tests for the smart indent feature: for pressing return, tab and space, especially in relation with `{` and `}`.
- [ ] Rainbow parenthesis should be able to span multiple lines.
- [ ] Shell scripts with if/else/endif blocks that are commented out are highlighted wrong.
- [ ] Add `guessica` to the `ctrl-t` menu.
- [ ] Let `guessica` also set `pkgrel=1` if there was a new version.
- [ ] Rainbow parentheses should span multiple lines, for Clojure, Emacs Lisp etc.
- [ ] Quotestate Process can not recognize triple runes, like the previous
      previous rune is ", the previous rune is " and the current rune is ".
      The wrong argumens are passed to the function. Figure out why.
- [ ] Rewrite `insertRune`. Improve word-wrap related functionality.
- [ ] Ignore multiline comments within multiline comments.
- [ ] Also enable rainbow parenthesis for lines that ends with a single-line comment.
- [ ] Introduce a type for screen cordinates, a type for screen coordinates + scroll offset, and another type for data coordinates.
- [ ] Insert a custom file from the command menu.
- [ ] Add one or more of these commands: regex search,
      go to definition, rename symbol, find references and disassembly.
- [ ] Make it easy to make recordings of the editing process.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Be able to edit `.txt.gz` and `.1.gz` files.
- [ ] Reduce memory usage.
- [ ] When in "SuggestMode", typing should start filtering the list.
- [ ] Highlight links in Markdown (perhaps color `[` and `]` yellow).
- [ ] Localization.
- [ ] If typing "dd" at the start or end of a line, delete it.
- [ ] If typing ":wq" at the start or end of a line, remove the text, save and quit.
- [ ] If typing ":w" at the start or end of a line, remove the text and save.
- [ ] If pressing return at the end of the document, after a full screen, then also scroll down 1 line.
      Currently, blank lines at the end of the document is immediately trimmed, which might make sense.
- [ ] Introduce a hexedit mode that will:
      * Not load the entire file into memory.
      * Display all bytes as a grid of "0xff" style fields, with the string representation to the right.
      * This might be better solved by having a separate hex editor?
- [ ] Replace and/or rewrite the syntax highlighting system.
- [ ] Plugins. When there's `txt2something` and `something2txt`, o should be able to edit "something" files in general.
      This could be used for hex editing, editing ELF files etc.
- [ ] Tab in the middle of a line, especially on a `|` character, could insert spaces until the `|` alignes with the `|` above, if applicable
      (For Markdown tables).
- [ ] Smarter indentation for `}`. There are still a few cases where it's not too smart.
      Perhaps use the logic for tab-indenting for when dedenting `}`?
- [ ] If joining a line that starts with a single-line comment with a line below that also starts with a single line comment,
      remove the extra comment marker.
- [ ] `ctrl-f` and then `return` could jump to a location at least 10 lines away that has been most visited within the last 10
      minutes.
- [ ] If over a certain percentage of the characters are not unicode.Graphics, enter binary mode.
- [ ] Extract the functionality for searching a MessagePack file to a `mpgrep` utility, that has a `-B` flag (like `grep`).

Nothing here.
