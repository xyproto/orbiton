# TODO

## Bug fixes

- [ ] Always catch the ctrl-c signal.
- [ ] The editor functions relating to the end of the text, file and document needs more testing.
- [ ] Fix rune insertion at the end of the terminal width.
- [ ] Fix markdown syntax highlighting refresh when entering checkboxes.

## Features I see myself using straight away

- [ ] File locking.
- [ ] Detect ISO-8859-1 and convert the file to UTF-8 before opening.
- [ ] New menu command: "insert file".
- [ ] Spellcheck all comments that are in English. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] If `xclip` or `wl-clipboard` are not found when pasting, present a status message. Also check related env. vars.
- [ ] If a word over N letters is typed 1 letter differently from all the other instances in the current file: color it differently.
- [ ] Reduce memory usage.
- [ ] Also format JSON documents with `ctrl-w`.
- [ ] Auto-detect if a loaded file uses `\t` or 1, 2, 3, 4, or 8 spaces for indentation.
- [ ] Let the autocompletion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Let the cut/copy/paste line state be part of the editor state, because of undo.

## Features in general

- [ ] Add one or more of these commands: regex search, hex editor,
      go to definition, rename symbol, find references and disassembly.
- [ ] Make it easy to make recordings of the editing process.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Be able to edit `.txt.gz` and `.1.gz` files.
- [ ] When in "SuggestMode", typing should start filtering the list.
- [ ] Highlight links in Markdown (perhaps color `[` and `]` yellow).
- [ ] Localization.

## Bug fixes that might not be needed

- [ ] Also enable rainbow parenthesis for lines that ends with a single-line comment.

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
