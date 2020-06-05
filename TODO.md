# TODO

## Bug fixes

- [ ] Pressing `ctrl-v` to paste does not work across X/Wayland sessions. It would be nice to find a more general clipboard solution.
- [ ] Resizing the terminal requires a press on `Esc` afterwards.

## Features I see myself using straight away

- [ ] Add guessica to the command menu.
- [ ] Add word wrap with a custom line length to the command menu.
- [ ] Autocompletion of filenames if the previous rune is "/" and tab is pressed.
- [ ] Insert `# vim: ts=2 sw=2 et:` at the bottom when `ctrl-space` is pressed in a PKGBUILD file. Or add a command menu option for this.
- [ ] If a word over N letters is typed 1 letter differently from all the other instances in the current file: color it differently.
- [ ] File locking.
- [ ] Spellcheck all comments that are in English. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] Should be able to open any binary file and save it again, without replacements. Add a hex edit mode.
- [ ] Detect ISO-8859-1 and convert the file to UTF-8 before opening.
- [ ] If `xclip` or `wl-clipboard` are not found when pasting, present a status message. Also check related env. vars.
- [ ] Also format JSON documents with `ctrl-w`.
- [ ] Auto-detect if a loaded file uses `\t` or 1, 2, 3, 4, or 8 spaces for indentation.
- [ ] Let the autocompletion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Let the cut/copy/paste line state be part of the editor state, because of undo.
- [ ] For git commit text, highlight column 80 if the text crosses that boundry.

## Low priority bug fixes

- [ ] Don't color parentheses in comments.
- [ ] Rainbow parentheses that span multiple lines, for Clojure, Emacs Lisp etc.
- [ ] Quotestate Process can not recognize triple runes, like the previous
      previous rune is ", the previous rune is " and the current rune is ".
      The wrong argumens are passed to the function. Figure out why.
- [ ] Rewrite `insertRune`. Improve word-wrap related functionality.
- [ ] Ignore multiline comments within multiline comments.
- [x] Fix markdown syntax highlighting refresh when entering checkboxes (was an altrgr+space issue).
- [ ] Also enable rainbow parenthesis for lines that ends with a single-line comment.
- [x] Markdown syntax highlighting should highlight item text the same until either a blank line or a line with a list item prefix.

## Features in general

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

## Features that might not be needed

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

## Smart (too smart?) features

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

## Refactoring

- [ ] Extract the functionality for searching a MessagePack file to a `mpgrep` utility, that has a `-B` flag (like `grep`).

## Unsorted

Nothing here.
